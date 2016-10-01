package httphandlers // import "resenje.org/httphandlers"

import (
	"fmt"
	"net/http"
)

func MaxBodyBytesHandler(h http.Handler, size int64, body string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > size {
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			w.WriteHeader(413)
			fmt.Fprintln(w, body)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, size)

		if err := r.ParseMultipartForm(1024); r.Body == nil && err != nil {
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			w.WriteHeader(413)
			fmt.Fprintln(w, body)
			return
		}
		h.ServeHTTP(w, r)
	})
}
