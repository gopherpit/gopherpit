// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webutils

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/boltdb/bolt"
)

// DumpDBHandler writes all data from BoltDB database to a http.ResponseWriter.
func DumpDBHandler(w http.ResponseWriter, r *http.Request, db *bolt.DB) {
	if db == nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "0")
		return
	}
	db.View(func(tx *bolt.Tx) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, filename := filepath.Split(db.Path())
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(w)
		return err
	})
}
