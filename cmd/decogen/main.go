package main

import (
	"flag"
	"log"
	"strings"

	"github.com/clobucks/decogen/internal/config"
	"github.com/clobucks/decogen/internal/generator"
	"github.com/clobucks/decogen/internal/parser"
)

func main() {
	// Parse command-line flags
	interfaceName := flag.String("interface", "", "Name of the interface to generate decorators for")
	sourceFile := flag.String("source", "", "Source file containing the interface")
	decorators := flag.String("decorators", "retry", "Comma-separated list of decorators to generate (retry,cache,metrics)")
	outputFile := flag.String("output", "", "Output file for generated code")
	packageName := flag.String("package", "decorators", "Package name for generated code")
	configFile := flag.String("config", "", "Path to configuration file")

	flag.Parse()

	var cfg *config.Config
	var err error

	// Load configuration from file if specified
	if *configFile != "" {
		cfg, err = config.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	} else {
		// Validate required flags
		if *interfaceName == "" {
			log.Fatal("Interface name is required")
		}
		if *sourceFile == "" {
			log.Fatal("Source file is required")
		}
		if *outputFile == "" {
			log.Fatal("Output file is required")
		}

		// Create configuration from flags
		cfg, err = config.FromFlags(*interfaceName, *sourceFile, *decorators, *outputFile, *packageName)
		if err != nil {
			log.Fatalf("Failed to create configuration: %v", err)
		}
	}

	// Parse the interface
	log.Printf("Parsing interface %s from %s", cfg.Interface.Name, cfg.Interface.Source)
	interfaceModel, err := parser.ParseInterface(cfg.Interface.Source, cfg.Interface.Name)
	if err != nil {
		log.Fatalf("Failed to parse interface: %v", err)
	}

	log.Printf("Found interface with %d methods", len(interfaceModel.Methods))

	// Get decorator types from configuration
	decoratorTypes, err := cfg.GetDecoratorTypes()
	if err != nil {
		log.Fatalf("Failed to get decorator types: %v", err)
	}

	// Create generator
	gen, err := generator.NewGenerator()
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	// Generate code
	log.Printf("Generating %s decorators for %s", strings.Join(*decorators, ","), cfg.Interface.Name)
	err = gen.Generate(interfaceModel, decoratorTypes, cfg.Package, cfg.Output)
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	log.Printf("Successfully generated code to %s", cfg.Output)
}
