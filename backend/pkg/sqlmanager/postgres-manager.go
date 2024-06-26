package sqlmanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	pg_queries "github.com/nucleuscloud/neosync/backend/gen/go/db/dbschemas/postgresql"
	"github.com/nucleuscloud/neosync/backend/internal/nucleusdb"
	"golang.org/x/sync/errgroup"
)

type PostgresManager struct {
	querier pg_queries.Querier
	pool    pg_queries.DBTX
	close   func()
}

func (p *PostgresManager) GetDatabaseSchema(ctx context.Context) ([]*DatabaseSchemaRow, error) {
	dbSchemas, err := p.querier.GetDatabaseSchema(ctx, p.pool)
	if err != nil && !nucleusdb.IsNoRows(err) {
		return nil, err
	} else if err != nil && nucleusdb.IsNoRows(err) {
		return []*DatabaseSchemaRow{}, nil
	}
	result := []*DatabaseSchemaRow{}
	for _, row := range dbSchemas {
		var generatedType *string
		if row.GeneratedType != "" {
			generatedType = &row.GeneratedType
		}
		result = append(result, &DatabaseSchemaRow{
			TableSchema:            row.TableSchema,
			TableName:              row.TableName,
			ColumnName:             row.ColumnName,
			DataType:               row.DataType,
			ColumnDefault:          row.ColumnDefault,
			IsNullable:             row.IsNullable,
			CharacterMaximumLength: row.CharacterMaximumLength,
			NumericPrecision:       row.NumericPrecision,
			NumericScale:           row.NumericScale,
			OrdinalPosition:        row.OrdinalPosition,
			GeneratedType:          generatedType,
		})
	}
	return result, nil
}

// returns: {public.users: { id: struct{}{}, created_at: struct{}{}}}
func (p *PostgresManager) GetSchemaColumnMap(ctx context.Context) (map[string]map[string]*ColumnInfo, error) {
	dbSchemas, err := p.GetDatabaseSchema(ctx)
	if err != nil {
		return nil, err
	}
	result := getUniqueSchemaColMappings(dbSchemas)
	return result, nil
}

func (p *PostgresManager) GetTableConstraintsBySchema(ctx context.Context, schemas []string) (*TableConstraints, error) {
	if len(schemas) == 0 {
		return &TableConstraints{}, nil
	}
	rows, err := p.querier.GetTableConstraintsBySchema(ctx, p.pool, schemas)
	if err != nil && !nucleusdb.IsNoRows(err) {
		return nil, err
	} else if err != nil && nucleusdb.IsNoRows(err) {
		return &TableConstraints{}, nil
	}

	foreignKeyMap := map[string][]*ForeignConstraint{}
	primaryKeyMap := map[string][]string{}
	uniqueConstraintsMap := map[string][][]string{}
	for _, row := range rows {
		tableName := BuildTable(row.SchemaName, row.TableName)
		switch row.ConstraintType {
		case "f":
			if len(row.ConstraintColumns) != len(row.ForeignColumnNames) {
				return nil, fmt.Errorf("length of columns was not equal to length of foreign key cols: %d %d", len(row.ConstraintColumns), len(row.ForeignColumnNames))
			}
			if len(row.ConstraintColumns) != len(row.Notnullable) {
				return nil, fmt.Errorf("length of columns was not equal to length of not nullable cols: %d %d", len(row.ConstraintColumns), len(row.Notnullable))
			}

			foreignKeyMap[tableName] = append(foreignKeyMap[tableName], &ForeignConstraint{
				Columns:     row.ConstraintColumns,
				NotNullable: row.Notnullable,
				ForeignKey: &ForeignKey{
					Table:   BuildTable(row.ForeignSchemaName, row.ForeignTableName),
					Columns: row.ForeignColumnNames,
				},
			})
		case "p":
			if _, exists := primaryKeyMap[tableName]; !exists {
				primaryKeyMap[tableName] = []string{}
			}
			primaryKeyMap[tableName] = append(primaryKeyMap[tableName], dedupeSlice(row.ConstraintColumns)...)
		case "u":
			columns := dedupeSlice(row.ConstraintColumns)
			uniqueConstraintsMap[tableName] = append(uniqueConstraintsMap[tableName], columns)
		}
	}
	return &TableConstraints{
		ForeignKeyConstraints: foreignKeyMap,
		PrimaryKeyConstraints: primaryKeyMap,
		UniqueConstraints:     uniqueConstraintsMap,
	}, nil
}

func (p *PostgresManager) GetForeignKeyConstraints(ctx context.Context, schemas []string) ([]*ForeignKeyConstraintsRow, error) {
	if len(schemas) == 0 {
		return []*ForeignKeyConstraintsRow{}, nil
	}
	rows, err := p.querier.GetTableConstraintsBySchema(ctx, p.pool, schemas)
	if err != nil && !nucleusdb.IsNoRows(err) {
		return nil, err
	} else if err != nil && nucleusdb.IsNoRows(err) {
		return []*ForeignKeyConstraintsRow{}, nil
	}

	result := []*ForeignKeyConstraintsRow{}
	for _, row := range rows {
		if row.ConstraintType != "f" {
			continue
		}
		if len(row.ConstraintColumns) != len(row.ForeignColumnNames) {
			return nil, fmt.Errorf("length of columns was not equal to length of foreign key cols: %d %d", len(row.ConstraintColumns), len(row.ForeignColumnNames))
		}
		if len(row.ConstraintColumns) != len(row.Notnullable) {
			return nil, fmt.Errorf("length of columns was not equal to length of not nullable cols: %d %d", len(row.ConstraintColumns), len(row.Notnullable))
		}

		for idx, colname := range row.ConstraintColumns {
			fkcol := row.ForeignColumnNames[idx]
			notnullable := row.Notnullable[idx]

			result = append(result, &ForeignKeyConstraintsRow{
				SchemaName:        row.SchemaName,
				TableName:         row.TableName,
				ColumnName:        colname,
				IsNullable:        !notnullable,
				ConstraintName:    row.ConstraintName,
				ForeignSchemaName: row.ForeignSchemaName,
				ForeignTableName:  row.ForeignTableName,
				ForeignColumnName: fkcol,
			})
		}
	}
	return result, nil
}

// Key is schema.table value is list of tables that key depends on
func (p *PostgresManager) GetForeignKeyConstraintsMap(ctx context.Context, schemas []string) (map[string][]*ForeignConstraint, error) {
	if len(schemas) == 0 {
		return map[string][]*ForeignConstraint{}, nil
	}
	constraints, err := p.GetTableConstraintsBySchema(ctx, schemas)
	if err != nil {
		return nil, err
	}

	if constraints == nil {
		return map[string][]*ForeignConstraint{}, nil
	}

	return constraints.ForeignKeyConstraints, nil
}

func (p *PostgresManager) GetPrimaryKeyConstraints(ctx context.Context, schemas []string) ([]*PrimaryKey, error) {
	if len(schemas) == 0 {
		return []*PrimaryKey{}, nil
	}
	rows, err := p.querier.GetTableConstraintsBySchema(ctx, p.pool, schemas)
	if err != nil && !nucleusdb.IsNoRows(err) {
		return nil, err
	} else if err != nil && nucleusdb.IsNoRows(err) {
		return []*PrimaryKey{}, nil
	}

	constraints := []*pg_queries.GetTableConstraintsBySchemaRow{}
	for _, row := range rows {
		if row.ConstraintType != "p" {
			continue
		}
		constraints = append(constraints, row)
	}
	result := []*PrimaryKey{}
	for _, row := range constraints {
		columns := dedupeSlice(row.ConstraintColumns)
		result = append(result, &PrimaryKey{
			Schema:  row.SchemaName,
			Table:   row.TableName,
			Columns: columns,
		})
	}
	return result, nil
}

func (p *PostgresManager) GetPrimaryKeyConstraintsMap(ctx context.Context, schemas []string) (map[string][]string, error) {
	if len(schemas) == 0 {
		return map[string][]string{}, nil
	}
	constraints, err := p.GetTableConstraintsBySchema(ctx, schemas)
	if err != nil {
		return nil, err
	}

	if constraints == nil {
		return map[string][]string{}, nil
	}

	return constraints.PrimaryKeyConstraints, nil
}

func (p *PostgresManager) GetUniqueConstraintsMap(ctx context.Context, schemas []string) (map[string][][]string, error) {
	if len(schemas) == 0 {
		return map[string][][]string{}, nil
	}
	constraints, err := p.GetTableConstraintsBySchema(ctx, schemas)
	if err != nil {
		return nil, err
	}

	if constraints == nil {
		return map[string][][]string{}, nil
	}

	return constraints.UniqueConstraints, nil
}

func (p *PostgresManager) GetRolePermissionsMap(ctx context.Context, role string) (map[string][]string, error) {
	rows, err := p.querier.GetPostgresRolePermissions(ctx, p.pool, role)
	if err != nil && !nucleusdb.IsNoRows(err) {
		return nil, err
	} else if err != nil && nucleusdb.IsNoRows(err) {
		return map[string][]string{}, nil
	}

	schemaTablePrivsMap := map[string][]string{}
	for _, permission := range rows {
		key := BuildTable(permission.TableSchema, permission.TableName)
		schemaTablePrivsMap[key] = append(schemaTablePrivsMap[key], permission.PrivilegeType)
	}
	return schemaTablePrivsMap, err
}

func (p *PostgresManager) GetCreateTableStatement(ctx context.Context, schema, table string) (string, error) {
	errgrp, errctx := errgroup.WithContext(ctx)

	var tableSchemas []*pg_queries.GetDatabaseTableSchemaRow
	errgrp.Go(func() error {
		result, err := p.querier.GetDatabaseTableSchema(errctx, p.pool, &pg_queries.GetDatabaseTableSchemaParams{
			Schema: schema,
			Table:  table,
		})
		if err != nil {
			return fmt.Errorf("unable to generate database table schema: %w", err)
		}
		tableSchemas = result
		return nil
	})
	var tableConstraints []*pg_queries.GetTableConstraintsRow
	errgrp.Go(func() error {
		result, err := p.querier.GetTableConstraints(errctx, p.pool, &pg_queries.GetTableConstraintsParams{
			Schema: schema,
			Table:  table,
		})
		if err != nil {
			return fmt.Errorf("unable to generate table constraints: %w", err)
		}
		tableConstraints = result
		return nil
	})
	if err := errgrp.Wait(); err != nil {
		return "", err
	}

	return generateCreateTableStatement(
		schema,
		table,
		tableSchemas,
		tableConstraints,
	), nil
}

// This assumes that the schemas and constraints as for a single table, not an entire db schema
func generateCreateTableStatement(
	schema string,
	table string,
	tableSchemas []*pg_queries.GetDatabaseTableSchemaRow,
	tableConstraints []*pg_queries.GetTableConstraintsRow,
) string {
	columns := make([]string, len(tableSchemas))
	for idx := range tableSchemas {
		record := tableSchemas[idx]
		columns[idx] = buildTableCol(record)
	}

	constraints := make([]string, len(tableConstraints))
	for idx := range tableConstraints {
		constraint := tableConstraints[idx]
		constraints[idx] = fmt.Sprintf("CONSTRAINT %s %s", constraint.ConstraintName, constraint.ConstraintDefinition)
	}
	tableDefs := append(columns, constraints...) //nolint:gocritic
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %q.%q (%s);`, schema, table, strings.Join(tableDefs, ", "))
}

func buildTableCol(record *pg_queries.GetDatabaseTableSchemaRow) string {
	pieces := []string{EscapePgColumn(record.ColumnName), buildDataType(record), buildNullableText(record)}
	if record.ColumnDefault != "" {
		if strings.HasPrefix(record.ColumnDefault, "nextval") && record.DataType == "integer" {
			pieces[1] = "SERIAL"
		} else if strings.HasPrefix(record.ColumnDefault, "nextval") && record.DataType == "bigint" {
			pieces[1] = "BIGSERIAL"
		} else if strings.HasPrefix(record.ColumnDefault, "nextval") && record.DataType == "smallint" {
			pieces[1] = "SMALLSERIAL"
		} else if record.ColumnDefault != "NULL" {
			pieces = append(pieces, "DEFAULT", record.ColumnDefault)
		}
	}
	return strings.Join(pieces, " ")
}

func buildDataType(record *pg_queries.GetDatabaseTableSchemaRow) string {
	return record.DataType
}

func buildNullableText(record *pg_queries.GetDatabaseTableSchemaRow) string {
	if record.IsNullable == "NO" {
		return "NOT NULL"
	}
	return "NULL"
}

func (p *PostgresManager) BatchExec(ctx context.Context, batchSize int, statements []string, opts *BatchExecOpts) error {
	for i := 0; i < len(statements); i += batchSize {
		end := i + batchSize
		if end > len(statements) {
			end = len(statements)
		}

		batchCmd := strings.Join(statements[i:end], "\n")
		if opts != nil && opts.Prefix != nil && *opts.Prefix != "" {
			batchCmd = fmt.Sprintf("%s %s", *opts.Prefix, batchCmd)
		}

		_, err := p.pool.Exec(ctx, batchCmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresManager) Exec(ctx context.Context, statement string) error {
	_, err := p.pool.Exec(ctx, statement)
	if err != nil {
		return err
	}
	return nil
}

func (p *PostgresManager) Close() {
	if p.pool != nil && p.close != nil {
		p.close()
	}
}

func BuildPgTruncateStatement(
	tables []string,
) string {
	return fmt.Sprintf("TRUNCATE TABLE %s;", strings.Join(tables, ", "))
}

func BuildPgTruncateCascadeStatement(
	schema string,
	table string,
) (string, error) {
	builder := goqu.Dialect("postgres")
	sqltable := goqu.S(schema).Table(table)
	stmt, _, err := builder.From(sqltable).Truncate().Cascade().ToSQL()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s;", stmt), nil
}

func EscapePgColumns(cols []string) []string {
	outcols := make([]string, len(cols))
	for idx := range cols {
		outcols[idx] = EscapePgColumn(cols[idx])
	}
	return outcols
}

func EscapePgColumn(col string) string {
	return fmt.Sprintf("%q", col)
}
