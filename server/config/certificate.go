// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"resenje.org/marshal"
)

// CertificateOptions defines parameters related to service's core functionality.
type CertificateOptions struct {
	DirectoryURL        string           `json:"directory-url" envconfig:"DIRECTORY_URL"`
	DirectoryURLStaging string           `json:"directory-url-staging" envconfig:"DIRECTORY_URL_STAGING"`
	RenewPeriod         marshal.Duration `json:"renew-period" envconfig:"RENEW_PERIOD"`
	RenewCheckPeriod    marshal.Duration `json:"renew-check-period" envconfig:"RENEW_CHECK_PERIOD"`
}

// NewCertificateOptions initializes CertificateOptions with default values.
func NewCertificateOptions() *CertificateOptions {
	return &CertificateOptions{
		DirectoryURL:        "https://acme-v01.api.letsencrypt.org/directory",
		DirectoryURLStaging: "https://acme-staging.api.letsencrypt.org/directory",
		RenewPeriod:         marshal.Duration(20 * 24 * time.Hour),
		RenewCheckPeriod:    marshal.Duration(23 * time.Hour),
	}
}

// Update updates options by loading certificate.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *CertificateOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "certificate.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_certificate", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// Verify checks if configuration values are valid and if all requirements are
// set for service to start.
func (o *CertificateOptions) Verify() (help string, err error) {
	return
}

// String returns a JSON representation of the options.
func (o *CertificateOptions) String() string {
	data, _ := json.MarshalIndent(o, "", "    ")
	return string(data)
}

// Prepare creates configured directories for home, storage, logs and
// temporary files.
func (o *CertificateOptions) Prepare() error {
	return nil
}
