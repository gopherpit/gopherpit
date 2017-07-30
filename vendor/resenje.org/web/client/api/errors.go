// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

// Error represents an error that contains a message and an error code.
// If the error is based on HTTP response status, message is status text
// and code status code.
type Error struct {
	// Message is a text that describes an error.
	Message string `json:"message"`
	// Code is a number that identifies error.
	// It allows error identification when serialization is involved.
	Code int `json:"code"`
}

// Error returns a Message value.
func (e *Error) Error() string {
	return e.Message
}
