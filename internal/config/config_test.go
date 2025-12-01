package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		_, err := Load("nonexistent.yaml")
		if err != ErrConfigNotFound {
			t.Errorf("expected ErrConfigNotFound, got %v", err)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "rum.yaml")
		os.WriteFile(path, []byte("invalid: [yaml"), 0644)

		_, err := Load(path)
		if err == nil {
			t.Error("expected error for invalid yaml")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "rum.yaml")

		content := `
templates:
  root: "."
  package: "main"
  dirs:
    - "templates/**/*.tmpl"
`
		os.WriteFile(path, []byte(content), 0644)

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Templates == nil {
			t.Fatal("expected templates config")
		}
		if cfg.Templates.Root != "." {
			t.Errorf("expected root '.', got %q", cfg.Templates.Root)
		}
		if cfg.Templates.Package != "main" {
			t.Errorf("expected package 'main', got %q", cfg.Templates.Package)
		}
		if len(cfg.Templates.Dirs) != 1 {
			t.Errorf("expected 1 dir, got %d", len(cfg.Templates.Dirs))
		}
	})

	t.Run("empty config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "rum.yaml")
		os.WriteFile(path, []byte(""), 0644)

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.HasTemplates() {
			t.Error("expected HasTemplates() to be false")
		}
	})
}

func TestHasTemplates(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "nil templates",
			config: Config{Templates: nil},
			want:   false,
		},
		{
			name:   "empty dirs",
			config: Config{Templates: &TemplatesConfig{Dirs: []string{}}},
			want:   false,
		},
		{
			name: "with dirs",
			config: Config{Templates: &TemplatesConfig{
				Dirs: []string{"templates/**/*.tmpl"},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasTemplates(); got != tt.want {
				t.Errorf("HasTemplates() = %v, want %v", got, tt.want)
			}
		})
	}
}
