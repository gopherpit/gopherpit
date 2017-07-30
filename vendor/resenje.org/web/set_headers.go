// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

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

// NewSetHeadersHandler sets provied headers on HTTP response.
func NewSetHeadersHandler(h http.Handler, headers map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, value := range headers {
			w.Header().Set(header, value)
		}
		h.ServeHTTP(w, r)
	})
}

// NoCacheHeadersHandler sets HTTP headers:
//     Cache-Control: no-cache, no-store, must-revalidate
//     Pragma: no-cache
//     Expires: 0
func NoCacheHeadersHandler(h http.Handler) http.Handler {
	return NewSetHeadersHandler(h, noCacheHeaders)
}

// NoExpireHeadersHandler sets HTTP headers:
//     Cache-Control: max-age=31536000
//     Expires: Thu, 31 Dec 2037 23:55:55 GMT
func NoExpireHeadersHandler(h http.Handler) http.Handler {
	return NewSetHeadersHandler(h, noExpireHeaders)
}
