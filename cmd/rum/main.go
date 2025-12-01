package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/4Sigma/rum/internal/config"
	"github.com/4Sigma/rum/internal/generator"
)

var (
	version = "dev"
	cfgFile string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "rum",
	Short:   "Rum - Code generation toolkit and http framework for Go projects",
	Long:    `Rum is a code generation toolkit and http framework that helps you generate boilerplate code for templates, services, repositories, and more.`,
	Version: version,
}

var genCmd = &cobra.Command{
	Use:     "gen",
	Aliases: []string{"generate"},
	Short:   "Generate code from rum.yaml configuration",
	Long: `Generate code based on the rum.yaml configuration file.

This command reads the rum.yaml file in the current directory and generates
code for all configured components (templates, services, etc.).

Example rum.yaml:

  templates:
    root: "."                        # where templates_gen.go is generated
    package: "main"                  # package name
    dirs:
      - "templates/**/*.tmpl"        # recursive glob pattern

Example structure:
  myproject/
  ├── templates/
  │   ├── openapi/
  │   │   └── api.yaml.tmpl
  │   └── pages/
  │       └── home.html.tmpl
  ├── templates_gen.go               # generated
  └── rum.yaml

Usage with go:generate:
  Add this comment to any Go file:
  //go:generate rum gen
`,
	RunE: runGenerate,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new rum.yaml configuration file",
	Long:  `Create a new rum.yaml configuration file with example settings.`,
	RunE:  runInit,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "rum.yaml", "config file path")
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(initCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	generated := false

	// Generate templates if configured
	if cfg.HasTemplates() {
		fmt.Println("Generating templates...")
		gen := generator.NewTemplatesGenerator(cfg.Templates)
		if err := gen.Generate(); err != nil {
			return fmt.Errorf("generating templates: %w", err)
		}
		generated = true
	}

	// Future: Add other generators here
	// if cfg.HasServices() { ... }
	// if cfg.HasRepositories() { ... }

	if !generated {
		fmt.Println("No components configured in rum.yaml. Nothing to generate.")
		fmt.Println("Run 'rum init' to create a sample configuration.")
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(cfgFile); err == nil {
		return fmt.Errorf("%s already exists", cfgFile)
	}

	sample := `# Rum configuration file
# Documentation: https://github.com/4Sigma/rum

# Template generation configuration
templates:
  # Root directory where templates_gen.go will be generated
  root: "."
  # Package name for generated code
  package: "main"
  # Template directories (glob patterns, supports **)
  dirs:
    - "templates/**/*.tmpl"

# Future components (not yet implemented):
# services:
#   output_dir: "internal/services"
# repositories:
#   output_dir: "internal/repositories"
#   sqlc_config: "sqlc.yaml"
# graphql:
#   schema: "schema.graphql"
# openapi:
#   spec: "openapi.yaml"
`

	if err := os.WriteFile(cfgFile, []byte(sample), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("Created %s\n", cfgFile)
	fmt.Println("Edit the file to configure your project, then run 'rum gen'")
	return nil
}
