package httphandlers // import "resenje.org/httphandlers"

import "net/http"

var (
	noCacheHeaders = map[string]string{
		"Cache-Control": "no-cache, no-store, must-revalidate",
		"Pragma":        "no-cache",
		"Expires":       "0",
	}
	noExpireHeaders = map[string]string{
		"Cache-Control": "max-age=31536000",
		"Expires":       "Thu, 31 Dec 2037 23:55:55 GMT",
	}
)

func SetHeadersHandler(h http.Handler, headers *map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, value := range *headers {
			w.Header().Set(header, value)
		}
		h.ServeHTTP(w, r)
	})
}

func NoCacheHeadersHandler(h http.Handler) http.Handler {
	return SetHeadersHandler(h, &noCacheHeaders)
}

func NoExpireHeadersHandler(h http.Handler) http.Handler {
	return SetHeadersHandler(h, &noExpireHeaders)
}
