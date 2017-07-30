// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"strings"
)

// NewStaticFilesHandler serves a file under specified filesystem if it
// can be opened, otherwise it serves HTTP from a specified handler.
func NewStaticFilesHandler(h http.Handler, prefix string, fs http.FileSystem) http.Handler {
	fileserver := http.StripPrefix(prefix, http.FileServer(fs))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, prefix)
		_, err := fs.Open(filename)
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		fileserver.ServeHTTP(w, r)
	})
}
