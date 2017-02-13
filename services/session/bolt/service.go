// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package boltSession provides a Service that is using local BoltDB database
// to store Session data.
package boltSession // import "gopherpit.com/gopherpit/services/session/bolt"

import (
	"os"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/session"
)

var (
	mmapFlags int
)

// Logger defines interface for logging messages with various severity levels.
type Logger interface {
	Debug(a ...interface{})
	Debugf(format string, a ...interface{})
	Info(a ...interface{})
	Errorf(format string, a ...interface{})
}

// Service implements gopherpit.com/gopherpit/services/session.Service interface.
type Service struct {
	DB *bolt.DB

	// DefaultLifetime is a period how long a session is valid if
	// no MaxTime is provided.
	DefaultLifetime time.Duration
	// CleanupPeriod defines a period on which deletion of expired
	// session is executed.
	CleanupPeriod time.Duration
	Logger        Logger
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

// Session retrieves a Session instance from a BoltDB database.
func (s Service) Session(id string) (*session.Session, error) {
	var r *sessionRecord
	if err := s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getSessionRecord(tx, []byte(id))
		return
	}); err != nil {
		return nil, err
	}
	return r.export(), nil
}

// CreateSession creates a new Session in a BoltDB database.
func (s Service) CreateSession(o *session.Options) (*session.Session, error) {
	r := &sessionRecord{}
	if o != nil {
		r.update(*o)
	}
	if err := s.DB.Update(func(tx *bolt.Tx) error {
		return r.save(tx, s.DefaultLifetime)
	}); err != nil {
		return nil, err
	}
	return r.export(), nil
}

// UpdateSession changes data of an existing Session.
func (s Service) UpdateSession(id string, o *session.Options) (*session.Session, error) {
	r := &sessionRecord{}
	if err := s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getSessionRecord(tx, []byte(id))
		if err != nil {
			return
		}
		if o != nil {
			r.update(*o)
		}
		return r.save(tx, s.DefaultLifetime)
	}); err != nil {
		return nil, err
	}
	return r.export(), nil
}

// DeleteSession deletes an existing Session.
func (s Service) DeleteSession(id string) error {
	return s.DB.Update(func(tx *bolt.Tx) error {
		return delete([]byte(id), tx)
	})
}
