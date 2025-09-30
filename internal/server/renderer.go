package server

import (
	"fmt"
	"html/template"
	"io"
	"strconv"
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
			// s = strings.TrimRight(s, "0")
			// s = strings.TrimRight(s, ".")
			return strings.ToUpper(currency) + " $" + s
		},
		"formatAmountString": func(amountStr string, currency string) string {
			if amountStr == "" {
				return currency + "0.00"
			}

			// Remove currency prefix if present (e.g., "USD21925" -> "21925")
			cleaned := strings.TrimSpace(amountStr)
			for _, prefix := range []string{"USD", "CAD", "EUR", "GBP", "$"} {
				if strings.HasPrefix(strings.ToUpper(cleaned), strings.ToUpper(prefix)) {
					cleaned = strings.TrimSpace(cleaned[len(prefix):])
					break
				}
			}

			// Try to parse as decimal first (e.g., "219.25")
			if strings.Contains(cleaned, ".") {
				if amount, err := strconv.ParseFloat(cleaned, 64); err == nil {
					return currency + " " + sprintf("%.2f", amount)
				}
			}

			// Try to parse as integer (assume it's already in cents)
			if amount, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
				major := float64(amount) / 100.0
				return currency + " " + sprintf("%.2f", major)
			}

			// If parsing fails, return the original string
			return currency + " " + amountStr
		},
	}
	t := template.Must(template.New("").Funcs(funcs).ParseGlob("templates/*.html"))
	return &TemplateRenderer{t: t}
}

func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.t.ExecuteTemplate(w, name, data)
}

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }
