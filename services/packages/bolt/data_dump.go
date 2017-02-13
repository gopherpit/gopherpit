// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltPackages

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/pkg/data-dump"
)

// DataDump implements dataDump.Interface interface to extract
// database data in a safe and reliable way.
func (s Service) DataDump(ifModifiedSince *time.Time) (dump *dataDump.Dump, err error) {
	if s.DB == nil {
		return
	}

	var stat os.FileInfo
	stat, err = os.Stat(s.DB.Path())
	if err != nil {
		return
	}
	modTime := stat.ModTime()
	if ifModifiedSince != nil && ifModifiedSince.After(modTime) {
		return
	}

	r, w := io.Pipe()
	var length int64

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				s.Logger.Errorf("packages service database dump: %s", err)
			}
		}()
		if err := s.DB.View(func(tx *bolt.Tx) error {
			length = tx.Size()
			wg.Done()
			_, err := tx.WriteTo(w)
			return err
		}); err != nil {
			panic(err)
		}
		w.Close()
	}()
	wg.Wait()

	_, name := filepath.Split(s.DB.Path())
	dump = &dataDump.Dump{
		Name:        name,
		ContentType: "application/octet-stream",
		Length:      length,
		ModTime:     &modTime,
		Body:        r,
	}
	return
}
