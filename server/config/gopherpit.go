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

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
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
	XSRFCookieName         string            `json:"xsrf-cookie-name" yaml:"xsrf-cookie-name" envconfig:"XSRF_COOKIE_NAME"`
	XSRFHeader             string            `json:"xsrf-header" yaml:"xsrf-header" envconfig:"XSRF_HEADER"`
	XSRFFormField          string            `json:"xsrf-form-field" yaml:"xsrf-form-field" envconfig:"XSRF_FORM_FIELD"`
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
		XSRFCookieName:         "secid",
		XSRFHeader:             "X-Secid",
		XSRFFormField:          "secid",
		Debug:                  false,
		PidFileName:            filepath.Join(BaseDir, Name+".pid"),
		PidFileMode:            0644,
		StorageFileMode:        0644,
		StorageDir:             filepath.Join(BaseDir, "storage"),
		AssetsDir:              filepath.Join(BaseDir, "assets"),
		StaticDir:              filepath.Join(BaseDir, "static"),
		TemplatesDir:           filepath.Join(BaseDir, "templates"),
		MaintenanceFilename:    "maintenance",
		GoogleAnalyticsID:      "",
		ContactRecipientEmail:  Name + "@localhost",
		SkipDomainVerification: true,
		VerificationSubdomain:  "_gopherpit",
		ForbiddenDomains:       []string{},
	}
}

// Update updates options by loading gopherpit.json files.
func (o *GopherPitOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "gopherpit.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "gopherpit.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1), o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// Verify checks if configuration values are valid and if all requirements are
// set for service to start.
func (o *GopherPitOptions) Verify() (help string, err error) {
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
	if _, err = os.Stat(o.AssetsDir); os.IsNotExist(err) {
		err = fmt.Errorf("Assets directory %s does not exist", o.AssetsDir)
		return
	}
	if _, err = os.Stat(o.TemplatesDir); os.IsNotExist(err) {
		err = fmt.Errorf("Templates directory %s does not exist", o.TemplatesDir)
		return
	}
	ln, err := net.Listen("tcp", o.Listen)
	if err != nil {
		return
	}
	ln.Close()
	lnTLS, err := net.Listen("tcp", o.ListenTLS)
	if err != nil {
		return
	}
	lnTLS.Close()
	return
}

// String returns a JSON representation of the options.
func (o *GopherPitOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Prepare creates configured directories for home, storage, logs and
// temporary files.
func (o *GopherPitOptions) Prepare() error {
	for _, dir := range []string{
		o.StorageDir,
		filepath.Dir(o.PidFileName),
	} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}
