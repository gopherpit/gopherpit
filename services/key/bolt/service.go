// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package boltKey provides a Service that is using local BoltDB database
// to store Key data.
package boltKey // import "gopherpit.com/gopherpit/services/key/bolt"

import (
	"os"
	"time"

	"gopherpit.com/gopherpit/services/key"

	"github.com/boltdb/bolt"
)

var (
	mmapFlags int
)

// Logger defines interface for logging messages with various severity levels.
type Logger interface {
	Errorf(format string, a ...interface{})
}

// Service implements gopherpit.com/gopherpit/services/key.Service interface.
type Service struct {
	DB     *bolt.DB
	Logger Logger
}

// NewDB opens a new BoltDB database.
func NewDB(filename string, fileMode os.FileMode, boltOptions *bolt.Options) (db *bolt.DB, err error) {
	if boltOptions == nil {
		boltOptions = &bolt.Options{
			Timeout:   2 * time.Second,
			MmapFlags: mmapFlags,
		}
	}
	if fileMode == 0 {
		fileMode = 0640
	}
	db, err = bolt.Open(filename, fileMode, boltOptions)
	return
}

func (s Service) KeyByRef(ref string) (k *key.Key, err error) {
	var r *keyRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getKeyRecordByRef(tx, []byte(ref))
		return
	}); err != nil {
		return
	}
	k = r.export()
	return
}

func (s Service) KeyBySecret(secret string) (k *key.Key, err error) {
	var r *keyRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getKeyRecordBySecret(tx, []byte(secret))
		return
	}); err != nil {
		return
	}
	k = r.export()
	return
}

func (s Service) CreateKey(ref string, o *key.Options) (k *key.Key, err error) {
	r := &keyRecord{
		Ref: ref,
	}
	if o != nil {
		r.update(o)
	}
	if err := s.DB.Update(func(tx *bolt.Tx) (err error) {
		_, err = getKeyRecordByRef(tx, []byte(ref))
		switch err {
		case key.ErrKeyNotFound:
		case nil:
			return key.ErrKeyRefAlreadyExists
		default:
			return err
		}
		return r.save(tx, "")
	}); err != nil {
		return nil, err
	}
	k = r.export()
	return
}

func (s Service) UpdateKey(ref string, o *key.Options) (k *key.Key, err error) {
	var r *keyRecord
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getKeyRecordByRef(tx, []byte(ref))
		if err != nil {
			return
		}
		if o != nil {
			r.update(o)
		}
		return r.save(tx, "")
	}); err != nil {
		return
	}
	k = r.export()
	return
}

func (s Service) DeleteKey(ref string) error {
	return s.DB.Update(func(tx *bolt.Tx) error {
		return delete(tx, []byte(ref))
	})
}

func (s Service) RegenerateSecret(ref string) (secret string, err error) {
	err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		secret, err = regenerateSecret(tx, []byte(ref))
		return
	})
	return
}

func (s Service) Keys(startName string, limit int) (page key.KeysPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getKeys(tx, []byte(startName), limit)
		return
	})
	return
}
