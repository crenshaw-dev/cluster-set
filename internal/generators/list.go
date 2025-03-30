package generators

import (
	"encoding/json"
	"fmt"

	"github.com/crenshaw-dev/cluster-set/api/v1alpha1"
)

func GetParametersFromListGenerator(generator *v1alpha1.ListGenerator) ([]map[string]any, error) {
	paramsList := make([]map[string]any, len(generator.Elements))
	for i := range generator.Elements {
		params := map[string]any{}
		err := json.Unmarshal(generator.Elements[i].Raw, &params)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal elements at index %d: %w", i, err)
		}
		paramsList[i] = params
	}
	return paramsList, nil
}
