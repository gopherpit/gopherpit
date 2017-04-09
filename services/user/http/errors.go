// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpUser

import (
	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/user"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrUnauthorized                 = errorRegistry.MustAddMessageError(401, "Unauthorized")
	ErrUserNotFound                 = errorRegistry.MustAddMessageError(1000, "User Not Found")
	ErrSaltNotFound                 = errorRegistry.MustAddMessageError(1001, "Salt Not Found")
	ErrPasswordUsed                 = errorRegistry.MustAddMessageError(1002, "Password Already Used")
	ErrUsernameMissing              = errorRegistry.MustAddMessageError(1100, "Username Missing")
	ErrUsernameInvalid              = errorRegistry.MustAddMessageError(1101, "Username Invalid")
	ErrUsernameExists               = errorRegistry.MustAddMessageError(1102, "Username Exists")
	ErrEmailMissing                 = errorRegistry.MustAddMessageError(1200, "Email Missing")
	ErrEmailInvalid                 = errorRegistry.MustAddMessageError(1201, "Email Invalid")
	ErrEmailExists                  = errorRegistry.MustAddMessageError(1202, "Email Exists")
	ErrEmailChangeEmailNotAvaliable = errorRegistry.MustAddMessageError(1300, "Email Not Available For Change")
	ErrEmailValidateTokenNotFound   = errorRegistry.MustAddMessageError(1301, "Email Validation Token Not Found")
	ErrEmailValidateTokenInvalid    = errorRegistry.MustAddMessageError(1302, "Email Validation Token Invalid")
	ErrEmailValidateTokenExpired    = errorRegistry.MustAddMessageError(1303, "Email Validation Token Expired")
	ErrPasswordResetTokenNotFound   = errorRegistry.MustAddMessageError(1400, "Password Reset Token Not Found")
	ErrPasswordResetTokenExpired    = errorRegistry.MustAddMessageError(1401, "Password Reset Token Expired")
)

var errorMap = map[error]error{
	ErrUnauthorized:                 user.ErrUnauthorized,
	ErrUserNotFound:                 user.ErrUserNotFound,
	ErrSaltNotFound:                 user.ErrSaltNotFound,
	ErrPasswordUsed:                 user.ErrPasswordUsed,
	ErrUsernameMissing:              user.ErrUsernameMissing,
	ErrUsernameInvalid:              user.ErrUsernameInvalid,
	ErrUsernameExists:               user.ErrUsernameExists,
	ErrEmailMissing:                 user.ErrEmailMissing,
	ErrEmailInvalid:                 user.ErrEmailInvalid,
	ErrEmailExists:                  user.ErrEmailExists,
	ErrEmailChangeEmailNotAvaliable: user.ErrEmailChangeEmailNotAvaliable,
	ErrEmailValidateTokenNotFound:   user.ErrEmailValidateTokenNotFound,
	ErrEmailValidateTokenInvalid:    user.ErrEmailValidateTokenInvalid,
	ErrEmailValidateTokenExpired:    user.ErrEmailValidateTokenExpired,
	ErrPasswordResetTokenNotFound:   user.ErrPasswordResetTokenNotFound,
	ErrPasswordResetTokenExpired:    user.ErrPasswordResetTokenExpired,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
