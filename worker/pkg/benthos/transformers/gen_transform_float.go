
// Code generated by Neosync neosync_transformer_generator.go. DO NOT EDIT.
// source: transform_float.go

package transformers

import (
	"strings"
	"fmt"
	transformer_utils "github.com/nucleuscloud/neosync/worker/pkg/benthos/transformers/utils"
	"github.com/nucleuscloud/neosync/worker/pkg/rng"
	
)

type TransformFloat64 struct{}

type TransformFloat64Opts struct {
	randomizer     rng.Rand
	
	randomizationRangeMin float64
	randomizationRangeMax float64
	precision *int64
	scale *int64
}

func NewTransformFloat64() *TransformFloat64 {
	return &TransformFloat64{}
}

func NewTransformFloat64Opts(
	randomizationRangeMinArg *float64,
	randomizationRangeMaxArg *float64,
	precision *int64,
	scale *int64,
  seedArg *int64,
) (*TransformFloat64Opts, error) {
	randomizationRangeMin := float64(1)
	if randomizationRangeMinArg != nil {
		randomizationRangeMin = *randomizationRangeMinArg
	}
	
	randomizationRangeMax := float64(10000)
	if randomizationRangeMaxArg != nil {
		randomizationRangeMax = *randomizationRangeMaxArg
	}
	
	seed, err := transformer_utils.GetSeedOrDefault(seedArg)
  if err != nil {
    return nil, fmt.Errorf("unable to generate seed: %w", err)
	}
	
	return &TransformFloat64Opts{
		randomizationRangeMin: randomizationRangeMin,
		randomizationRangeMax: randomizationRangeMax,
		precision: precision,
		scale: scale,
		randomizer: rng.New(seed),	
	}, nil
}

func (o *TransformFloat64Opts) BuildBloblangString(
	valuePath string,	
) string {
	fnStr := []string{
		"value:this.%s", 
		"randomization_range_min:%v", 
		"randomization_range_max:%v",
	}

	params := []any{
		valuePath,
	 	o.randomizationRangeMin,
	 	o.randomizationRangeMax,
	}

	
	if o.precision != nil {
		fnStr = append(fnStr, "precision:%v")
		params = append(params, *o.precision)
	}
	if o.scale != nil {
		fnStr = append(fnStr, "scale:%v")
		params = append(params, *o.scale)
	}

	template := fmt.Sprintf("transform_float64(%s)", strings.Join(fnStr, ","))
	return fmt.Sprintf(template, params...)
}

func (t *TransformFloat64) GetJsTemplateData() (*TemplateData, error) {
	return &TemplateData{
		Name: "transformFloat64",
		Description: "Anonymizes and transforms an existing float value.",
		Example: "",
	}, nil
}

func (t *TransformFloat64) ParseOptions(opts map[string]any) (any, error) {
	transformerOpts := &TransformFloat64Opts{}

	randomizationRangeMin, ok := opts["randomizationRangeMin"].(float64)
	if !ok {
		randomizationRangeMin = 1
	}
	transformerOpts.randomizationRangeMin = randomizationRangeMin

	randomizationRangeMax, ok := opts["randomizationRangeMax"].(float64)
	if !ok {
		randomizationRangeMax = 10000
	}
	transformerOpts.randomizationRangeMax = randomizationRangeMax

	var precision *int64
	if arg, ok := opts["precision"].(int64); ok {
		precision = &arg
	}
	transformerOpts.precision = precision

	var scale *int64
	if arg, ok := opts["scale"].(int64); ok {
		scale = &arg
	}
	transformerOpts.scale = scale

	var seedArg *int64
	if seedValue, ok := opts["seed"].(int64); ok {
			seedArg = &seedValue
	}
	seed, err := transformer_utils.GetSeedOrDefault(seedArg)
	if err != nil {
		return nil, fmt.Errorf("unable to generate seed: %w", err)
	}
	transformerOpts.randomizer = rng.New(seed)

	return transformerOpts, nil
}