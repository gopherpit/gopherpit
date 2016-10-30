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

	"github.com/kelseyhightower/envconfig"
	"resenje.org/httputils/client/http"
)

// ServicesOptions defines parameters for communication with external services.
type ServicesOptions struct {
	UserEndpoint         string              `json:"user-endpoint" envconfig:"USER_ENDPOINT"`
	UserKey              string              `json:"user-key" envconfig:"USER_KEY"`
	UserOptions          *httpClient.Options `json:"user-options" envconfig:"USER_OPTIONS"`
	SessionEndpoint      string              `json:"session-endpoint" envconfig:"SESSION_ENDPOINT"`
	SessionKey           string              `json:"session-key" envconfig:"SESSION_KEY"`
	SessionOptions       *httpClient.Options `json:"session-options" envconfig:"SESSION_OPTIONS"`
	NotificationEndpoint string              `json:"notification-endpoint" envconfig:"NOTIFICATION_ENDPOINT"`
	NotificationKey      string              `json:"notification-key" envconfig:"NOTIFICATION_KEY"`
	NotificationOptions  *httpClient.Options `json:"notification-options" envconfig:"NOTIFICATION_OPTIONS"`
	CertificateEndpoint  string              `json:"certificate-endpoint" envconfig:"CERTIFICATE_ENDPOINT"`
	CertificateKey       string              `json:"certificate-key" envconfig:"CERTIFICATE_KEY"`
	CertificateOptions   *httpClient.Options `json:"certificate-options" envconfig:"CERTIFICATE_OPTIONS"`
	PackagesEndpoint     string              `json:"packages-endpoint" envconfig:"PACKAGES_ENDPOINT"`
	PackagesKey          string              `json:"packages-key" envconfig:"PACKAGES_KEY"`
	PackagesOptions      *httpClient.Options `json:"packages-options" envconfig:"PACKAGES_OPTIONS"`
}

// NewServicesOptions initializes ServicesOptions with empty values.
func NewServicesOptions() *ServicesOptions {
	return &ServicesOptions{}
}

// Update updates options by loading services.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *ServicesOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "services.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_services", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a JSON representation of the options.
func (o *ServicesOptions) String() string {
	data, _ := json.MarshalIndent(o, "", "    ")
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *ServicesOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *ServicesOptions) Prepare() error {
	return nil
}
