// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpNotification

import (
	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/notification"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrEmailAlreadySent = errorRegistry.MustAddMessageError(1000, "Email Already Sent")
)

var errorMap = map[error]error{
	ErrEmailAlreadySent: notification.ErrEmailAlreadySent,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
