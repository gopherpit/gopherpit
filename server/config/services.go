// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
	"resenje.org/httputils/client/http"
)

// ServicesOptions defines parameters for communication with external services.
type ServicesOptions struct {
	UserEndpoint         string              `json:"user-endpoint" yaml:"user-endpoint" envconfig:"USER_ENDPOINT"`
	UserKey              string              `json:"user-key" yaml:"user-key" envconfig:"USER_KEY"`
	UserOptions          *httpClient.Options `json:"user-options" yaml:"user-options" envconfig:"USER_OPTIONS"`
	SessionEndpoint      string              `json:"session-endpoint" yaml:"session-endpoint" envconfig:"SESSION_ENDPOINT"`
	SessionKey           string              `json:"session-key" yaml:"session-key" envconfig:"SESSION_KEY"`
	SessionOptions       *httpClient.Options `json:"session-options" yaml:"session-options" envconfig:"SESSION_OPTIONS"`
	NotificationEndpoint string              `json:"notification-endpoint" yaml:"notification-endpoint" envconfig:"NOTIFICATION_ENDPOINT"`
	NotificationKey      string              `json:"notification-key" yaml:"notification-key" envconfig:"NOTIFICATION_KEY"`
	NotificationOptions  *httpClient.Options `json:"notification-options" yaml:"notification-options" envconfig:"NOTIFICATION_OPTIONS"`
	CertificateEndpoint  string              `json:"certificate-endpoint" yaml:"certificate-endpoint" envconfig:"CERTIFICATE_ENDPOINT"`
	CertificateKey       string              `json:"certificate-key" yaml:"certificate-key" envconfig:"CERTIFICATE_KEY"`
	CertificateOptions   *httpClient.Options `json:"certificate-options" yaml:"certificate-options" envconfig:"CERTIFICATE_OPTIONS"`
	PackagesEndpoint     string              `json:"packages-endpoint" yaml:"packages-endpoint" envconfig:"PACKAGES_ENDPOINT"`
	PackagesKey          string              `json:"packages-key" yaml:"packages-key" envconfig:"PACKAGES_KEY"`
	PackagesOptions      *httpClient.Options `json:"packages-options" yaml:"packages-options" envconfig:"PACKAGES_OPTIONS"`
}

// NewServicesOptions initializes ServicesOptions with empty values.
func NewServicesOptions() *ServicesOptions {
	return &ServicesOptions{}
}

// Update updates options by loading services.json files.
func (o *ServicesOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "services.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "services.json")
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
	data, _ := yaml.Marshal(o)
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
