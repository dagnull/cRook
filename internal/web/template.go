package web

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

var templateMap map[string]*template.Template

// InitTemplates loads each page template into its own template.Template so that
// {{define "content"}} blocks in different pages don't overwrite each other.
func InitTemplates(uiFS fs.FS) {
	tmplFS, err := fs.Sub(uiFS, "ui/templates")
	if err != nil {
		panic("ui/templates sub-fs: " + err.Error())
	}

	pages := []string{"home.html", "room.html"}
	templateMap = make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t := template.Must(
			template.New("").ParseFS(tmplFS, "base.html", "partials/*.html", page),
		)
		templateMap[page] = t
	}
}

func renderTemplate(w http.ResponseWriter, name string, data any) {
	t, ok := templateMap[name]
	if !ok {
		slog.Error("template not found", "name", name)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The unnamed entry point in each page file is the {{template "base" .}} call.
	// Execute the named file template to kick it off.
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template render", "name", name, "err", err)
	}
}
