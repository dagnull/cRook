package web

import (
	"io/fs"
	"net/http"
)

func staticHandler(uiFS fs.FS) http.Handler {
	staticFS, err := fs.Sub(uiFS, "ui/static")
	if err != nil {
		panic("ui/static sub-fs: " + err.Error())
	}
	return http.FileServer(http.FS(staticFS))
}
