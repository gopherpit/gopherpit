package httphandlers // import "resenje.org/httphandlers"

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func MethodHandler(h map[string]http.Handler, body string, contentType string, w http.ResponseWriter, r *http.Request) {
	if handler, ok := h[r.Method]; ok {
		handler.ServeHTTP(w, r)
	} else {
		allow := []string{}
		for k := range h {
			allow = append(allow, k)
		}
		sort.Strings(allow)
		w.Header().Set("Allow", strings.Join(allow, ", "))
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintln(w, body)
		}
	}
}
