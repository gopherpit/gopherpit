// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func redirect(w http.ResponseWriter, r *http.Request, location string) {
	if q := r.URL.RawQuery; q != "" {
		location += "?" + q
	}
	w.Header().Set("Location", location)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusFound)
}

func open(root, name string) (http.File, error) {
	if root == "" {
		root = "."
	}
	return os.Open(filepath.Join(root, filepath.FromSlash(path.Clean("/"+name))))
}
