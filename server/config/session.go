// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"time"

	"resenje.org/marshal"
)

// SessionOptions defines parameters related to session storage.
type SessionOptions struct {
	CleanupPeriod   marshal.Duration `json:"cleanup-period" yaml:"cleanup-period" envconfig:"CLEANUP_PERIOD"`
	DefaultLifetime marshal.Duration `json:"default-lifetime" yaml:"default-lifetime" envconfig:"DEFAULT_LIFETIME"`
}

// NewSessionOptions initializes SessionOptions with default values.
func NewSessionOptions() *SessionOptions {
	return &SessionOptions{
		CleanupPeriod:   marshal.Duration(25 * time.Hour),
		DefaultLifetime: marshal.Duration(30 * 24 * time.Hour),
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *SessionOptions) VerifyAndPrepare() error { return nil }
