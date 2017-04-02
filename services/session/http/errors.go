// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpSession

import (
	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/session"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrSessionNotFound = errorRegistry.MustAddMessageError(1000, "Session Not Found")
)

var errorMap = map[error]error{
	ErrSessionNotFound: session.ErrSessionNotFound,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
