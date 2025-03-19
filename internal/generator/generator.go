package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/clobucks/decogen/internal/model"
)

// DecoratorType represents the type of decorator to generate
type DecoratorType string

const (
	// RetryDecorator generates a retry decorator
	RetryDecorator DecoratorType = "retry"
	// CacheDecorator generates a cache decorator
	CacheDecorator DecoratorType = "cache"
	// MetricsDecorator generates a metrics decorator
	MetricsDecorator DecoratorType = "metrics"
)

// Generator handles code generation for decorators
type Generator struct {
	templates map[DecoratorType]*template.Template
}

// NewGenerator creates a new generator with loaded templates
func NewGenerator() (*Generator, error) {
	g := &Generator{
		templates: make(map[DecoratorType]*template.Template),
	}

	// Load retry template
	retryTemplate, err := template.ParseFiles("internal/generator/templates/retry.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load retry template: %w", err)
	}
	g.templates[RetryDecorator] = retryTemplate

	// Load other templates as needed
	// ...

	return g, nil
}

// Generate generates code for the specified interface and decorators
func (g *Generator) Generate(
	interfaceModel *model.Interface,
	decoratorTypes []DecoratorType,
	outputPackage string,
	outputPath string,
) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate each decorator
	for _, dt := range decoratorTypes {
		tmpl, ok := g.templates[dt]
		if !ok {
			return fmt.Errorf("unknown decorator type: %s", dt)
		}

		// Prepare template data
		data := map[string]interface{}{
			"PackageName": outputPackage,
			"Name":        interfaceModel.Name,
			"Methods":     interfaceModel.Methods,
			"Imports":     interfaceModel.Imports,
			"Comments":    interfaceModel.Comments,
		}

		// Create a buffer for the generated code
		var buf strings.Builder

		// Execute the template
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		// Format the generated code
		formattedCode, err := format.Source([]byte(buf.String()))
		if err != nil {
			// If formatting fails, still write the unformatted code
			// so we can diagnose the issue
			if err := os.WriteFile(outputPath, []byte(buf.String()), 0644); err != nil {
				return fmt.Errorf("failed to write unformatted code: %w", err)
			}
			return fmt.Errorf("failed to format generated code: %w", err)
		}

		// Write the formatted code to the output file
		if err := os.WriteFile(outputPath, formattedCode, 0644); err != nil {
			return fmt.Errorf("failed to write generated code: %w", err)
		}
	}

	return nil
}
