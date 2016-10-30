// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import (
	"errors"
	"sync"
)

// ErrErrorAlreadyRegistered is returned an error with the same code
// is found in the error registry.
var ErrErrorAlreadyRegistered = errors.New("error already registered")

// ErrorRegistry defines an interface to retrieve error by a numerical code.
type ErrorRegistry interface {
	Error(int) error
}

// MapErrorRegistry uses map to store errors and their codes.
type MapErrorRegistry struct {
	errors map[int]error
	mu     *sync.Mutex
}

// NewMapErrorRegistry creates a new instance of MapErrorRegistry.
func NewMapErrorRegistry(errors map[int]error) *MapErrorRegistry {
	if errors == nil {
		errors = map[int]error{}
	}
	return &MapErrorRegistry{
		errors: errors,
		mu:     &sync.Mutex{},
	}
}

// AddError adds a new error with a code to the registry.
// It there already is an error with the same code,
// ErrErrorAlreadyRegistered will be returned.
func (r *MapErrorRegistry) AddError(code int, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.errors[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	r.errors[code] = err
	return nil
}

// Error returns an error that is registered under the provided code.
func (r MapErrorRegistry) Error(code int) error {
	return r.errors[code]
}
