
// Code generated by Neosync neosync_transformer_generator.go. DO NOT EDIT.
// source: generate_city.go

package transformers

import (
	"strings"
	"fmt"
	transformer_utils "github.com/nucleuscloud/neosync/worker/pkg/benthos/transformers/utils"
	"github.com/nucleuscloud/neosync/worker/pkg/rng"
	
)

type GenerateCity struct{}

type GenerateCityOpts struct {
	randomizer     rng.Rand
	
	maxLength int64
}

func NewGenerateCity() *GenerateCity {
	return &GenerateCity{}
}

func NewGenerateCityOpts(
	maxLengthArg *int64,
  seedArg *int64,
) (*GenerateCityOpts, error) {
	maxLength := int64(100)
	if maxLengthArg != nil {
		maxLength = *maxLengthArg
	}
	
	seed, err := transformer_utils.GetSeedOrDefault(seedArg)
  if err != nil {
    return nil, fmt.Errorf("unable to generate seed: %w", err)
	}
	
	return &GenerateCityOpts{
		maxLength: maxLength,
		randomizer: rng.New(seed),	
	}, nil
}

func (o *GenerateCityOpts) BuildBloblangString(	
) string {
	fnStr := []string{ 
		"max_length:%v",
	}

	params := []any{
	 	o.maxLength,
	}

	

	template := fmt.Sprintf("generate_city(%s)", strings.Join(fnStr, ","))
	return fmt.Sprintf(template, params...)
}

func (t *GenerateCity) GetJsTemplateData() (*TemplateData, error) {
	return &TemplateData{
		Name: "generateCity",
		Description: "Randomly selects a city from a list of predefined US cities.",
		Example: "",
	}, nil
}

func (t *GenerateCity) ParseOptions(opts map[string]any) (any, error) {
	transformerOpts := &GenerateCityOpts{}

	maxLength, ok := opts["maxLength"].(int64)
	if !ok {
		maxLength = 100
	}
	transformerOpts.maxLength = maxLength

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