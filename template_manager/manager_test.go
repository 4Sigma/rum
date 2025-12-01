package rumtpl

import (
	"testing"
	"testing/fstest"
)

func TestNewManagerFromFS(t *testing.T) {
	fs := fstest.MapFS{
		"templates/home.html.tmpl":  {Data: []byte("Hello {{.Name}}")},
		"templates/about.html.tmpl": {Data: []byte("About page")},
		"other/skip.txt":            {Data: []byte("skip this")},
	}

	m, err := NewManagerFromFS(fs, "*.tmpl")
	if err != nil {
		t.Fatalf("NewManagerFromFS error: %v", err)
	}

	// Should find templates
	if m.t.Lookup("templates/home.html.tmpl") == nil {
		t.Error("expected to find home.html.tmpl")
	}
	if m.t.Lookup("templates/about.html.tmpl") == nil {
		t.Error("expected to find about.html.tmpl")
	}
}

func TestRender(t *testing.T) {
	fs := fstest.MapFS{
		"home.html.tmpl": {Data: []byte("Hello {{.Name}}")},
	}

	m, err := NewManagerFromFS(fs, "*.tmpl")
	if err != nil {
		t.Fatalf("NewManagerFromFS error: %v", err)
	}

	data := map[string]string{"Name": "World"}
	result, err := m.Render("home.html.tmpl", data)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	expected := "Hello World"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRenderNotFound(t *testing.T) {
	fs := fstest.MapFS{
		"home.html.tmpl": {Data: []byte("Hello")},
	}

	m, err := NewManagerFromFS(fs, "*.tmpl")
	if err != nil {
		t.Fatalf("NewManagerFromFS error: %v", err)
	}

	_, err = m.Render("notfound.tmpl", nil)
	if err != ErrTemplateError {
		t.Errorf("expected ErrTemplateError, got %v", err)
	}
}

func TestRenderWithPath(t *testing.T) {
	// Test con path completo (come viene generato da rum gen)
	fs := fstest.MapFS{
		"templates/pages/home.html.tmpl": {Data: []byte("Page: {{.Title}}")},
	}

	m, err := NewManagerFromFS(fs, "*.tmpl")
	if err != nil {
		t.Fatalf("NewManagerFromFS error: %v", err)
	}

	data := map[string]string{"Title": "Home"}
	result, err := m.Render("templates/pages/home.html.tmpl", data)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	expected := "Page: Home"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}
