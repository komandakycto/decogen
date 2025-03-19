package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clobucks/decogen/internal/generator"
)

// Config represents the configuration for the decorator generator
type Config struct {
	// Interface configuration
	Interface struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"interface"`

	// Decorators to generate
	Decorators []struct {
		Name   string                 `json:"name"`
		Config map[string]interface{} `json:"config"`
	} `json:"decorators"`

	// Output configuration
	Output  string `json:"output"`
	Package string `json:"package"`

	// Additional imports
	Imports []string `json:"imports"`
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetDecoratorTypes converts string decorator names to DecoratorType values
func (c *Config) GetDecoratorTypes() ([]generator.DecoratorType, error) {
	var types []generator.DecoratorType

	for _, dec := range c.Decorators {
		switch strings.ToLower(dec.Name) {
		case "retry":
			types = append(types, generator.RetryDecorator)
		case "cache":
			types = append(types, generator.CacheDecorator)
		case "metrics":
			types = append(types, generator.MetricsDecorator)
		default:
			return nil, fmt.Errorf("unknown decorator type: %s", dec.Name)
		}
	}

	return types, nil
}

// FromFlags creates a configuration from command-line flags
func FromFlags(
	interfaceName string,
	sourcePath string,
	decoratorsStr string,
	outputPath string,
	packageName string,
) (*Config, error) {
	config := &Config{}
	config.Interface.Name = interfaceName
	config.Interface.Source = sourcePath
	config.Output = outputPath
	config.Package = packageName

	// Parse decorators string (comma-separated)
	if decoratorsStr != "" {
		decoratorNames := strings.Split(decoratorsStr, ",")
		for _, name := range decoratorNames {
			config.Decorators = append(config.Decorators, struct {
				Name   string                 `json:"name"`
				Config map[string]interface{} `json:"config"`
			}{
				Name:   strings.TrimSpace(name),
				Config: make(map[string]interface{}),
			})
		}
	}

	return config, nil
}
