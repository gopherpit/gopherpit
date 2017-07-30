// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// HandleMethods uses a corresponding Handler based on HTTP request method.
// If Handler is not found, a method not allowed HTTP response is returned
// with specified body and Content-Type header.
func HandleMethods(methods map[string]http.Handler, body string, contentType string, w http.ResponseWriter, r *http.Request) {
	if handler, ok := methods[r.Method]; ok {
		handler.ServeHTTP(w, r)
	} else {
		allow := []string{}
		for k := range methods {
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
