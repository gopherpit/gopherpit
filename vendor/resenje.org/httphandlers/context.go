package httphandlers // import "resenje.org/httphandlers"

import (
	"net/http"

	"golang.org/x/net/context"
)

type ContextResponseWriter struct {
	http.ResponseWriter
	context.Context
}

func (r *ContextResponseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *ContextResponseWriter) CloseNotify() <-chan bool {
	if f, ok := r.ResponseWriter.(http.CloseNotifier); ok {
		return f.CloseNotify()
	}
	return make(<-chan bool)
}

func (r *ContextResponseWriter) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func ContextHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&ContextResponseWriter{w, context.Background()}, r)
	})
}
