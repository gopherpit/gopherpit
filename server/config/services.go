// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"resenje.org/web/client/http"
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
	KeyEndpoint          string              `json:"key-endpoint" yaml:"key-endpoint" envconfig:"KEY_ENDPOINT"`
	KeyKey               string              `json:"key-key" yaml:"key-key" envconfig:"KEY_KEY"`
	KeyOptions           *httpClient.Options `json:"key-options" yaml:"key-options" envconfig:"KEY_OPTIONS"`
	GCRAStoreEndpoint    string              `json:"gcra-store-endpoint" yaml:"gcra-store-endpoint" envconfig:"GCRASTORE_ENDPOINT"`
	GCRAStoreKey         string              `json:"gcra-store-key" yaml:"gcra-store-key" envconfig:"GCRASTORE_KEY"`
	GCRAStoreOptions     *httpClient.Options `json:"gcra-store-options" yaml:"gcra-store-options" envconfig:"GCRASTORE_OPTIONS"`
}

// NewServicesOptions initializes ServicesOptions with empty values.
func NewServicesOptions() *ServicesOptions {
	return &ServicesOptions{}
}

// VerifyAndPrepare implements application.Options interface.
func (o *ServicesOptions) VerifyAndPrepare() error { return nil }
