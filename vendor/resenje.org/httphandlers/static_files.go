package httphandlers // import "resenje.org/httphandlers"

import (
	"net/http"
	"strings"
)

func StaticFilesHandler(h http.Handler, prefix string, fs http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, prefix)
		_, err := fs.Open(filename)
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		fileserver := http.StripPrefix(prefix, http.FileServer(fs))
		fileserver.ServeHTTP(w, r)
	})
}
