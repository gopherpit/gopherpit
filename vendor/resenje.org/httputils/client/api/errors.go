// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

// Error represents an error that contains a message and an error code.
// If the error is based on HTTP response status, message is status text
// and code status code.
type Error struct {
	Message string
	Code    int
}

func (e *Error) Error() string {
	return e.Message
}
