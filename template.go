package mofu

import "io"

// TemplateEngine renders templates.
type TemplateEngine interface {
	Render(w io.Writer, name string, data any) error
}

// TemplateConfig configures template engine.
type TemplateConfig interface {
	CreateEngine() (TemplateEngine, error)
}
