package rumtpl

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"path/filepath"
)

var (
	ErrTemplateError = errors.New("template error")
)

// Renderer is the minimal interface consumers use.
type Renderer interface {
	Render(Name, any) ([]byte, error)
}

// Name type for template identifier.
type Name string

// Manager holds parsed templates.
type Manager struct{ t *template.Template }

// NewManagerFromFS parses templates from any fs.FS matching pattern.
func NewManagerFromFS(fsys fs.FS, pattern string) (*Manager, error) {
	t := template.New("rum")
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if match, _ := filepath.Match(pattern, filepath.Base(path)); !match {
			return nil
		}

		b, rerr := fs.ReadFile(fsys, path)
		if rerr != nil {
			return rerr
		}
		_, perr := t.New(filepath.Base(path)).Parse(string(b))
		return perr
	})

	if err != nil {
		return nil, err
	}
	return &Manager{t: t}, nil
}

// NewManagerFromEmbed convenience when package embeds templates in subdir.
func NewManagerFromEmbed(f embed.FS, subdir, pattern string) (*Manager, error) {
	s, err := fs.Sub(f, subdir)
	if err != nil {
		return nil, err
	}
	return NewManagerFromFS(s, pattern)
}

// Render implements Renderer.
func (m *Manager) Render(name Name, data any) ([]byte, error) {
	var buf bytes.Buffer
	t := m.t.Lookup(string(name))
	if t == nil {
		return nil, ErrTemplateError
	}
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
