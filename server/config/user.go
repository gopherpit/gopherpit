// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

// UserOptions defines parameters related to the user management.
type UserOptions struct {
	RememberMeDays        int `json:"remember-me-days" yaml:"remember-me-days" envconfig:"REMEMBER_ME_DAYS"`
	PasswordNoReuseMonths int `json:"password-no-reuse-months" yaml:"password-no-reuse-months" envconfig:"PASSWORD_NO_REUSE_MONTHS"`
}

// NewUserOptions initializes UserOptions with default values.
func NewUserOptions() *UserOptions {
	return &UserOptions{
		RememberMeDays:        45,
		PasswordNoReuseMonths: 0,
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *UserOptions) VerifyAndPrepare() error { return nil }
