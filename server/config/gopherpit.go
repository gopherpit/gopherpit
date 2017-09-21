// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"resenje.org/marshal"
)

// GopherPitOptions defines parameters related to service's core functionality.
type GopherPitOptions struct {
	Listen                 string            `json:"listen" yaml:"listen" envconfig:"LISTEN"`
	ListenTLS              string            `json:"listen-tls" yaml:"listen-tls" envconfig:"LISTEN_TLS"`
	ListenInternal         string            `json:"listen-internal" yaml:"listen-internal" envconfig:"LISTEN_INTERNAL"`
	ListenInternalTLS      string            `json:"listen-internal-tls" yaml:"listen-internal-tls" envconfig:"LISTEN_INTERNAL_TLS"`
	TLSCert                string            `json:"tls-cert" yaml:"tls-cert" envconfig:"TLS_CERT"`
	TLSKey                 string            `json:"tls-key" yaml:"tls-key" envconfig:"TLS_KEY"`
	Brand                  string            `json:"brand" yaml:"brand" envconfig:"BRAND"`
	Domain                 string            `json:"domain" yaml:"domain" envconfig:"DOMAIN"`
	Headers                map[string]string `json:"headers" yaml:"headers" envconfig:"HEADERS"`
	SessionCookieName      string            `json:"session-cookie-name" yaml:"session-cookie-name" envconfig:"SESSION_COOKIE_NAME"`
	Debug                  bool              `json:"debug" yaml:"debug" envconfig:"DEBUG"`
	PidFileName            string            `json:"pid-file" yaml:"pid-file" envconfig:"PID_FILE"`
	PidFileMode            marshal.Mode      `json:"pid-file-mode" yaml:"pid-file-mode" envconfig:"PID_FILE_MODE"`
	StorageFileMode        marshal.Mode      `json:"storage-file-mode" yaml:"storage-file-mode" envconfig:"STORAGE_FILE_MODE"`
	StorageDir             string            `json:"storage-dir" yaml:"storage-dir" envconfig:"STORAGE_DIR"`
	AssetsDir              string            `json:"assets-dir" yaml:"assets-dir" envconfig:"ASSETS_DIR"`
	StaticDir              string            `json:"static-dir" yaml:"static-dir" envconfig:"STATIC_DIR"`
	TemplatesDir           string            `json:"templates-dir" yaml:"templates-dir" envconfig:"TEMPLATES_DIR"`
	MaintenanceFilename    string            `json:"maintenance-filename" yaml:"maintenance-filename" envconfig:"MAINTENANCE_FILENAME"`
	GoogleAnalyticsID      string            `json:"google-analytics-id" yaml:"google-analytics-id" envconfig:"GOOGLE_ANALYTICS_ID"`
	ContactRecipientEmail  string            `json:"contact-recipient-email" yaml:"contact-recipient-email" envconfig:"CONTACT_RECIPIENT_EMAIL"`
	SkipDomainVerification bool              `json:"skip-domain-verification" yaml:"skip-domain-verification" envconfig:"SKIP_DOMAIN_VERIFICATION"`
	VerificationSubdomain  string            `json:"verification-subdomain" yaml:"verification-subdomain" envconfig:"VERIFICATION_SUBDOMAIN"`
	TrustedDomains         []string          `json:"trusted-domains" yaml:"trusted-domains" envconfig:"TRUSTED_DOMAINS"`
	ForbiddenDomains       []string          `json:"forbidden-domains" yaml:"forbidden-domains" envconfig:"FORBIDDEN_DOMAINS"`
}

// NewGopherPitOptions initializes GopherPitOptions with default values.
func NewGopherPitOptions() *GopherPitOptions {
	return &GopherPitOptions{
		Listen:            ":8080",
		ListenTLS:         "",
		ListenInternal:    "",
		ListenInternalTLS: "",
		TLSCert:           "",
		TLSKey:            "",
		Brand:             "GopherPit",
		Domain:            "localhost",
		Headers: map[string]string{
			"Server":           Name + "/" + Version + "-" + BuildInfo,
			"X-Frame-Options":  "SAMEORIGIN",
			"X-XSS-Protection": "1; mode=block",
		},
		SessionCookieName:      "sesid",
		Debug:                  false,
		PidFileName:            filepath.Join(BaseDir, Name+".pid"),
		PidFileMode:            0644,
		StorageFileMode:        0644,
		StorageDir:             filepath.Join(BaseDir, "storage"),
		AssetsDir:              "",
		StaticDir:              "",
		TemplatesDir:           "",
		MaintenanceFilename:    "maintenance",
		GoogleAnalyticsID:      "",
		ContactRecipientEmail:  Name + "@localhost",
		SkipDomainVerification: false,
		VerificationSubdomain:  "_gopherpit",
		TrustedDomains:         []string{},
		ForbiddenDomains:       []string{},
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *GopherPitOptions) VerifyAndPrepare() (err error) {
	if o.TLSCert != "" {
		if !strings.HasPrefix(o.TLSCert, "/") {
			o.TLSCert = filepath.Join(BaseDir, o.TLSCert)
		}
		if _, err = os.Open(o.TLSCert); err != nil {
			err = fmt.Errorf("%s: %s", err, o.TLSCert)
			return
		}
	}
	if o.TLSKey != "" {
		if !strings.HasPrefix(o.TLSKey, "/") {
			o.TLSKey = filepath.Join(BaseDir, o.TLSKey)
		}
		if _, err = os.Open(o.TLSKey); err != nil {
			err = fmt.Errorf("%s: %s", err, o.TLSKey)
			return
		}
	}
	if o.AssetsDir != "" {
		if _, err = os.Stat(o.AssetsDir); os.IsNotExist(err) {
			err = fmt.Errorf("Assets directory %s does not exist", o.AssetsDir)
			return
		}
	}
	if o.StaticDir != "" {
		if _, err = os.Stat(o.StaticDir); os.IsNotExist(err) {
			err = fmt.Errorf("Static directory %s does not exist", o.TemplatesDir)
			return
		}
	}
	if o.TemplatesDir != "" {
		if _, err = os.Stat(o.TemplatesDir); os.IsNotExist(err) {
			err = fmt.Errorf("Templates directory %s does not exist", o.TemplatesDir)
			return
		}
	}
	ln, err := net.Listen("tcp", o.Listen)
	if err != nil {
		return
	}
	ln.Close()
	ln, err = net.Listen("tcp", o.ListenTLS)
	if err != nil {
		return
	}
	ln.Close()
	ln, err = net.Listen("tcp", o.ListenInternal)
	if err != nil {
		return
	}
	ln.Close()
	ln, err = net.Listen("tcp", o.ListenInternalTLS)
	if err != nil {
		return
	}
	ln.Close()

	for _, dir := range []string{
		o.StorageDir,
		filepath.Dir(o.PidFileName),
	} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0777); err != nil {
				return err
			}
		}
	}
	return
}
