// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpKey

import (
	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/key"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrKeyNotFound         = errorRegistry.MustAddMessageError(1000, "Key Not Found")
	ErrKeyRefAlreadyExists = errorRegistry.MustAddMessageError(1001, "Key Reference Already Exists")
	ErrKeyRefRequired      = errorRegistry.MustAddMessageError(1002, "Key Reference Required")
)

var errorMap = map[error]error{
	ErrKeyNotFound:         key.ErrKeyNotFound,
	ErrKeyRefAlreadyExists: key.ErrKeyRefAlreadyExists,
	ErrKeyRefRequired:      key.ErrKeyRefRequired,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
