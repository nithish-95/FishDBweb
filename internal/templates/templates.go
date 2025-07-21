package templates

import (
	"html/template"
)

var (
	Templates *template.Template
)

// Initialize templates with custom functions
func InitTemplates() error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"lt":  func(a, b int) bool { return a < b },
		"gt":  func(a, b int) bool { return a > b },
	}

	tmpl := template.New("").Funcs(funcMap)
	var err error
	Templates, err = tmpl.ParseGlob("templates/*.html")
	return err
}
