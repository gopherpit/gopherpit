// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import "errors"

// ErrErrorAlreadyRegistered is returned an error with the same code
// is found in the error registry.
var ErrErrorAlreadyRegistered = errors.New("error already registered")

// ErrorRegistry defines an interface to retrieve error by a numerical code.
type ErrorRegistry interface {
	Error(int) error
	Handler(int) func(body []byte) error
}

// MapErrorRegistry uses map to store errors and their codes.
// It is assumed that adding of errors will be performed on
// initialization of program and therefore it is not locked
// for concurrent writes. Concurrent reads are safe as Go maps
// allow it. If concurrent adding and reading of errors in registry
// is needed, an implementation with locks must be used.
type MapErrorRegistry struct {
	errors   map[int]error
	handlers map[int]func(body []byte) error
}

// NewMapErrorRegistry creates a new instance of MapErrorRegistry.
func NewMapErrorRegistry(errors map[int]error, handlers map[int]func(body []byte) error) *MapErrorRegistry {
	if errors == nil {
		errors = map[int]error{}
	}
	if handlers == nil {
		handlers = map[int]func(body []byte) error{}
	}
	return &MapErrorRegistry{
		errors:   errors,
		handlers: handlers,
	}
}

// AddError adds a new error with a code to the registry.
// It there already is an error or handler with the same code,
// ErrErrorAlreadyRegistered will be returned.
func (r *MapErrorRegistry) AddError(code int, err error) error {
	if _, ok := r.errors[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	if _, ok := r.handlers[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	r.errors[code] = err
	return nil
}

// AddMessageError adds a new Error isntance with a code and message
// to the registry.
// It there already is an error or handler with the same code,
// ErrErrorAlreadyRegistered will be returned.
func (r *MapErrorRegistry) AddMessageError(code int, message string) (*Error, error) {
	if _, ok := r.errors[code]; ok {
		return nil, ErrErrorAlreadyRegistered
	}
	if _, ok := r.handlers[code]; ok {
		return nil, ErrErrorAlreadyRegistered
	}
	err := &Error{
		Message: message,
		Code:    code,
	}
	r.errors[code] = err
	return err, nil
}

// MustAddError calls AddError and panics in case of an error.
func (r *MapErrorRegistry) MustAddError(code int, err error) {
	if e := r.AddError(code, err); e != nil {
		panic(e)
	}
}

// MustAddMessageError calls AddMessageError and panics in case of an error.
func (r *MapErrorRegistry) MustAddMessageError(code int, message string) *Error {
	err, e := r.AddMessageError(code, message)
	if e != nil {
		panic(e)
	}
	return err
}

// Error returns an error that is registered under the provided code.
func (r MapErrorRegistry) Error(code int) error {
	return r.errors[code]
}

// AddHandler adds a new error handler with a code to the registry.
// It there already is an error or handler with the same code,
// ErrErrorAlreadyRegistered will be returned.
func (r *MapErrorRegistry) AddHandler(code int, handler func(body []byte) error) error {
	if _, ok := r.errors[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	if _, ok := r.handlers[code]; ok {
		return ErrErrorAlreadyRegistered
	}
	r.handlers[code] = handler
	return nil
}

// MustAddHandler calls AddHandler and panics in case of an error.
func (r *MapErrorRegistry) MustAddHandler(code int, handler func(body []byte) error) {
	if err := r.AddHandler(code, handler); err != nil {
		panic(err)
	}
}

// Handler returns a handler that is registered under the provided code.
func (r MapErrorRegistry) Handler(code int) func(body []byte) error {
	return r.handlers[code]
}
