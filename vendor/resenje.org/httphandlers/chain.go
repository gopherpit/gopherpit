package httphandlers // import "resenje.org/httphandlers"

import "net/http"

func ChainHandlers(handlers ...func(http.Handler) http.Handler) (handler http.Handler) {
	for i := len(handlers) - 1; i >= 0; i-- {
		handler = handlers[i](handler)
	}
	return
}
