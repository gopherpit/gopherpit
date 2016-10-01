// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package user

import "fmt"

var (
	// ErrorRegistry is a map of error codes to errors.
	// It is usually used in gopherpit.com/gopherpit/pkg/client.Client.
	ErrorRegistry = map[int]error{}
	serviceName   = "user"
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
	if _, ok := ErrorRegistry[code]; ok {
		panic(fmt.Sprintf("%s service error %v registered twice", serviceName, code))
	}
	ErrorRegistry[code] = err
	return
}
