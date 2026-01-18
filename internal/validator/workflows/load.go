package workflows

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a workflow definition from YAML.
func Load(path string) (Definition, error) {
	var def Definition
	data, err := os.ReadFile(path)
	if err != nil {
		return def, err
	}
	if err := yaml.Unmarshal(data, &def); err != nil {
		return def, err
	}
	return def, nil
}
