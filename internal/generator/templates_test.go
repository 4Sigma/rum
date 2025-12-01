package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/4Sigma/rum/internal/config"
)

func TestPathToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"templates/openapi/api.yaml.tmpl", "OpenapiApi"},
		{"templates/pages/home.html.tmpl", "PagesHome"},
		{"templates/user-profile.html.tmpl", "UserProfile"},
		{"templates/emails/welcome_email.html.tmpl", "EmailsWelcomeEmail"},
		{"template/simple.tmpl", "Simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pathToPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("pathToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitRecursivePattern(t *testing.T) {
	tests := []struct {
		pattern     string
		wantBase    string
		wantPattern string
	}{
		{"templates/**/*.tmpl", "templates", "*.tmpl"},
		{"**/*.tmpl", ".", "*.tmpl"},
		{"src/views/**/*.html.tmpl", "src/views", "*.html.tmpl"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			base, pat := splitRecursivePattern(tt.pattern)
			if base != tt.wantBase {
				t.Errorf("base = %q, want %q", base, tt.wantBase)
			}
			if pat != tt.wantPattern {
				t.Errorf("pattern = %q, want %q", pat, tt.wantPattern)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	// Setup test directory
	dir := t.TempDir()

	// Create template structure
	templatesDir := filepath.Join(dir, "templates")
	pagesDir := filepath.Join(templatesDir, "pages")
	emailsDir := filepath.Join(templatesDir, "emails")

	os.MkdirAll(pagesDir, 0755)
	os.MkdirAll(emailsDir, 0755)

	// Create templates
	os.WriteFile(filepath.Join(pagesDir, "home.html.tmpl"), []byte("{{.Title}}"), 0644)
	os.WriteFile(filepath.Join(emailsDir, "welcome.html.tmpl"), []byte("Hello {{.Name}}"), 0644)

	// Configure generator
	cfg := &config.TemplatesConfig{
		Root:    dir,
		Package: "main",
		Dirs:    []string{"templates/**/*.tmpl"},
	}

	gen := NewTemplatesGenerator(cfg)
	err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify output file
	outputFile := filepath.Join(dir, "templates_gen.go")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	output := string(content)

	// Check package
	if !strings.Contains(output, "package main") {
		t.Error("expected 'package main' in output")
	}

	// Check constants
	if !strings.Contains(output, "PagesHome") {
		t.Error("expected 'PagesHome' constant")
	}
	if !strings.Contains(output, "EmailsWelcome") {
		t.Error("expected 'EmailsWelcome' constant")
	}

	// Check paths in constants
	if !strings.Contains(output, `"templates/pages/home.html.tmpl"`) {
		t.Error("expected full path in constant value")
	}

	// Check embed directive
	if !strings.Contains(output, "//go:embed") {
		t.Error("expected embed directive")
	}

	// Check init function
	if !strings.Contains(output, "func init()") {
		t.Error("expected init function")
	}
}

func TestGenerateNoTemplates(t *testing.T) {
	dir := t.TempDir()

	// Create empty templates dir
	os.MkdirAll(filepath.Join(dir, "templates"), 0755)

	cfg := &config.TemplatesConfig{
		Root:    dir,
		Package: "main",
		Dirs:    []string{"templates/**/*.tmpl"},
	}

	gen := NewTemplatesGenerator(cfg)
	err := gen.Generate()

	if err == nil {
		t.Error("expected error for empty templates")
	}
	if !strings.Contains(err.Error(), "no templates found") {
		t.Errorf("expected 'no templates found' error, got: %v", err)
	}
}

func TestGenerateInvalidTemplate(t *testing.T) {
	dir := t.TempDir()

	templatesDir := filepath.Join(dir, "templates")
	os.MkdirAll(templatesDir, 0755)

	// Invalid template syntax
	os.WriteFile(filepath.Join(templatesDir, "bad.html.tmpl"), []byte("{{.Invalid"), 0644)

	cfg := &config.TemplatesConfig{
		Root:    dir,
		Package: "main",
		Dirs:    []string{"templates/*.tmpl"},
	}

	gen := NewTemplatesGenerator(cfg)
	err := gen.Generate()

	if err == nil {
		t.Error("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestGenerateDuplicateNames(t *testing.T) {
	dir := t.TempDir()

	// Create two dirs with same filename
	dir1 := filepath.Join(dir, "templates", "a")
	dir2 := filepath.Join(dir, "templates", "b")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	// Same base name, different content - should create different const names
	os.WriteFile(filepath.Join(dir1, "test.html.tmpl"), []byte("A"), 0644)
	os.WriteFile(filepath.Join(dir2, "test.html.tmpl"), []byte("B"), 0644)

	cfg := &config.TemplatesConfig{
		Root:    dir,
		Package: "main",
		Dirs:    []string{"templates/**/*.tmpl"},
	}

	gen := NewTemplatesGenerator(cfg)
	err := gen.Generate()

	// Should work since paths are different (ATest vs BTest)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
