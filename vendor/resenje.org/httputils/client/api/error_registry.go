// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import (
	"errors"
	"sync"
)

var ErrErrorAlreadyRegistered = errors.New("error already registered")

// ErrorRegistry defines an interface to retrieve error by it's numerical code.
type ErrorRegistry interface {
	Error(int) error
}

type MapErrorRegistry struct {
	errors map[int]error
	mu     *sync.Mutex
}

func NewMapErrorRegistry(errors map[int]error) *MapErrorRegistry {
	if errors == nil {
		errors = map[int]error{}
	}
	return &MapErrorRegistry{
		errors: errors,
		mu:     &sync.Mutex{},
	}
}

func (r *MapErrorRegistry) AddError(code int, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.errors[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	r.errors[code] = err
	return nil
}

func (r MapErrorRegistry) Error(code int) error {
	return r.errors[code]
}
