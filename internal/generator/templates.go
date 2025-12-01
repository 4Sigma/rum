package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/4Sigma/rum/internal/config"
)

// TemplateInfo holds information about a discovered template.
type TemplateInfo struct {
	FileName  string // Original filename: "api.template.yaml.tmpl"
	RelPath   string // Relative path from root: "templates/openapi/api.template.yaml.tmpl"
	ConstName string // PascalCase name with path prefix: "OpenapiApiTemplate"
}

// TemplatesGenerator generates Go code for template management.
type TemplatesGenerator struct {
	config *config.TemplatesConfig
}

// NewTemplatesGenerator creates a new template generator.
func NewTemplatesGenerator(cfg *config.TemplatesConfig) *TemplatesGenerator {
	return &TemplatesGenerator{config: cfg}
}

// Generate scans template sources and generates the output file.
func (g *TemplatesGenerator) Generate() error {
	var allTemplates []TemplateInfo
	seenNames := make(map[string]string) // constName -> relPath for duplicate detection

	for _, dir := range g.config.Dirs {
		templates, err := g.scanDir(dir)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", dir, err)
		}

		// Check for duplicates
		for _, t := range templates {
			if existing, ok := seenNames[t.ConstName]; ok {
				return fmt.Errorf("duplicate constant name %q from %q and %q", t.ConstName, existing, t.RelPath)
			}
			seenNames[t.ConstName] = t.RelPath
		}

		allTemplates = append(allTemplates, templates...)
	}

	if len(allTemplates) == 0 {
		return fmt.Errorf("no templates found in configured dirs")
	}

	// Validate templates syntax
	if err := g.validateTemplates(allTemplates); err != nil {
		return err
	}

	// Generate the output file
	return g.generateFile(allTemplates)
}

// scanDir scans a directory using glob pattern for template files.
func (g *TemplatesGenerator) scanDir(pattern string) ([]TemplateInfo, error) {
	var templates []TemplateInfo

	root := g.config.Root
	if root == "" {
		root = "."
	}

	// Handle recursive glob with **
	if strings.Contains(pattern, "**") {
		baseDir, filePattern := splitRecursivePattern(pattern)
		fullBaseDir := baseDir
		if root != "." {
			fullBaseDir = filepath.Join(root, baseDir)
		}

		err := filepath.WalkDir(fullBaseDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			matched, err := filepath.Match(filePattern, d.Name())
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)
			templates = append(templates, TemplateInfo{
				FileName:  d.Name(),
				RelPath:   relPath,
				ConstName: pathToPascalCase(relPath),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		fullPattern := pattern
		if root != "." {
			fullPattern = filepath.Join(root, pattern)
		}

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}

			relPath, _ := filepath.Rel(root, path)
			templates = append(templates, TemplateInfo{
				FileName:  filepath.Base(path),
				RelPath:   relPath,
				ConstName: pathToPascalCase(relPath),
			})
		}
	}

	return templates, nil
}

// splitRecursivePattern splits "templates/**/*.tmpl" into "templates" and "*.tmpl"
func splitRecursivePattern(pattern string) (baseDir, filePattern string) {
	idx := strings.Index(pattern, "**")
	if idx == -1 {
		return ".", pattern
	}

	baseDir = strings.TrimSuffix(pattern[:idx], "/")
	if baseDir == "" {
		baseDir = "."
	}

	remainder := pattern[idx+2:]
	remainder = strings.TrimPrefix(remainder, "/")
	if remainder == "" {
		filePattern = "*"
	} else {
		filePattern = remainder
	}

	return baseDir, filePattern
}

// validateTemplates checks template syntax by parsing them.
func (g *TemplatesGenerator) validateTemplates(templates []TemplateInfo) error {
	var errs []error

	root := g.config.Root
	if root == "" {
		root = "."
	}

	for _, t := range templates {
		fullPath := filepath.Join(root, t.RelPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("reading %s: %w", t.RelPath, err))
			continue
		}

		_, err = template.New(t.FileName).Parse(string(content))
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing %s: %w", t.RelPath, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("template validation failed:\n%v", errs)
	}
	return nil
}

// generateFile creates the generated Go file.
func (g *TemplatesGenerator) generateFile(templates []TemplateInfo) error {
	root := g.config.Root
	if root == "" {
		root = "."
	}

	outputFile := filepath.Join(root, "templates_gen.go")

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Collect unique directories for embed
	embedDirs := make(map[string]bool)
	for _, dir := range g.config.Dirs {
		// Convert pattern to embed-compatible format
		embedDir := strings.ReplaceAll(dir, "**", "*")
		embedDirs[embedDir] = true
	}

	var embedPatterns []string
	for dir := range embedDirs {
		embedPatterns = append(embedPatterns, dir)
	}

	data := struct {
		Package       string
		Templates     []TemplateInfo
		EmbedPatterns []string
		Dirs          []string
	}{
		Package:       g.config.Package,
		Templates:     templates,
		EmbedPatterns: embedPatterns,
		Dirs:          g.config.Dirs,
	}

	var buf bytes.Buffer
	if err := outputTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	fmt.Printf("Generated %s with %d templates\n", outputFile, len(templates))
	return nil
}

// pathToPascalCase converts a path like "templates/openapi/api.template.yaml.tmpl" to "OpenapiApiTemplate"
func pathToPascalCase(path string) string {
	// Remove common prefixes
	path = strings.TrimPrefix(path, "templates/")
	path = strings.TrimPrefix(path, "template/")

	// Remove extensions
	path = strings.TrimSuffix(path, ".tmpl")
	path = strings.TrimSuffix(path, ".html")
	path = strings.TrimSuffix(path, ".txt")
	path = strings.TrimSuffix(path, ".yaml")
	path = strings.TrimSuffix(path, ".json")
	path = strings.TrimSuffix(path, ".template")

	// Replace path separators and other separators with spaces
	re := regexp.MustCompile(`[-_./\\]`)
	path = re.ReplaceAllString(path, " ")

	// Title case each word and join
	words := strings.Fields(path)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

var outputTemplate = template.Must(template.New("output").Parse(`// Code generated by rum. DO NOT EDIT.
//go:generate rum gen

package {{.Package}}

import (
	"embed"

	rumtpl "github.com/4Sigma/rum/template_manager"
)

{{range .EmbedPatterns}}//go:embed {{.}}
{{end}}var templatesFS embed.FS

// TemplateName is a type-safe template identifier.
type TemplateName = rumtpl.Name

const (
{{- range .Templates}}
	{{.ConstName}} TemplateName = "{{.RelPath}}"
{{- end}}
)

// Manager is the template manager instance.
var Manager *rumtpl.Manager

func init() {
	var err error
	Manager, err = rumtpl.NewManagerFromFS(templatesFS, "*.tmpl")
	if err != nil {
		panic("rum: failed to initialize template manager: " + err.Error())
	}
}
`))
