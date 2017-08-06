// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"time"

	"resenje.org/marshal"
)

// CertificateOptions defines parameters related to service's core functionality.
type CertificateOptions struct {
	DirectoryURL        string           `json:"directory-url" yaml:"directory-url" envconfig:"DIRECTORY_URL"`
	DirectoryURLStaging string           `json:"directory-url-staging" yaml:"directory-url-staging" envconfig:"DIRECTORY_URL_STAGING"`
	RenewPeriod         marshal.Duration `json:"renew-period" yaml:"renew-period" envconfig:"RENEW_PERIOD"`
	RenewCheckPeriod    marshal.Duration `json:"renew-check-period" yaml:"renew-check-period" envconfig:"RENEW_CHECK_PERIOD"`
}

// NewCertificateOptions initializes CertificateOptions with default values.
func NewCertificateOptions() *CertificateOptions {
	return &CertificateOptions{
		DirectoryURL:        "https://acme-v01.api.letsencrypt.org/directory",
		DirectoryURLStaging: "https://acme-staging.api.letsencrypt.org/directory",
		RenewPeriod:         marshal.Duration(21 * 24 * time.Hour),
		RenewCheckPeriod:    marshal.Duration(23 * time.Hour),
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *CertificateOptions) VerifyAndPrepare() error { return nil }
