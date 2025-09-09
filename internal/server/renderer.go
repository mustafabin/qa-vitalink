package server

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	t *template.Template
}

func NewRenderer() *TemplateRenderer {
	funcs := template.FuncMap{
		"formatAmount": func(cents int64, currency string) string {
			major := float64(cents) / 100.0
			s := sprintf("%.2f", major)
			s = strings.TrimRight(s, "0")
			s = strings.TrimRight(s, ".")
			return strings.ToUpper(currency) + " " + s
		},
	}
	t := template.Must(template.New("").Funcs(funcs).ParseGlob("templates/*.html"))
	return &TemplateRenderer{t: t}
}

func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.t.ExecuteTemplate(w, name, data)
}

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }
