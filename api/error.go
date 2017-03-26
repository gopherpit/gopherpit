// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"

	"resenje.org/httputils/client/api"
)

var (
	// ErrorRegistry is a map of error codes to errors.
	ErrorRegistry = apiClient.NewMapErrorRegistry(nil)
)

// Error is a structure that holds error message and code.
type Error struct {
	// Message is a text that describes an error.
	Message string `json:"message"`
	// Code is a number that identifies error.
	// It allows error identification when serialization is involved.
	Code int `json:"code"`
}

// Error returns error message.
func (e *Error) Error() string {
	return e.Message
}

// NewError creates an instance of Error and adds it to ErrorRegistry.
// If error code already exists in ErrorRegistry, it panics.
func NewError(code int, message string) (err *Error) {
	err = &Error{
		Message: message,
		Code:    code,
	}
	if e := ErrorRegistry.AddError(code, err); e != nil {
		panic(fmt.Sprintf("error registry: code %v: %s", code, e))
	}
	return
}
