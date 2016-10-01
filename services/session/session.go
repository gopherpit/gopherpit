// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package session // import "gopherpit.com/gopherpit/services/session"

// Session holds session ID and values associated.
type Session struct {
	// A unique Session ID.
	ID string `json:"id"`
	// A mapping of named arbitrary data that Session holds.
	Values map[string]interface{} `json:"values,omitempty"`
	// MaxAge is an integer that represents duration in seconds and
	// can be used in setting expiration of a HTTP cookie.
	MaxAge int `json:"max-age,omitempty"`
}

// Options is a structure with parameters as pointers to set
// session data. If a parameter is nil, the corresponding Session
// parameter will not be changed.
type Options struct {
	Values *map[string]interface{} `json:"values,omitempty"`
	MaxAge *int                    `json:"max-age,omitempty"`
}

// Service defines functions that Session provider must have.
type Service interface {
	// Session retrieves a Session instance.
	Session(id string) (*Session, error)
	// CreateSession creates a new Session instance.
	CreateSession(*Options) (*Session, error)
	// UpdateSession changes data of an existing Session.
	UpdateSession(id string, o *Options) (*Session, error)
	// DeleteSession deletes an existing Session.
	DeleteSession(id string) error
}

var (
	// SessionNotFound is error if session does not exist.
	SessionNotFound = NewError(1000, "session not found")
)
