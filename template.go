package mofu

import "io"

type TemplateEngine interface {
	Render(w io.Writer, name string, data any) error
}

type TemplateConfig interface {
	CreateEngine() (TemplateEngine, error)
}
