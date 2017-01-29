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
)

// EmailOptions defines parameters for email sending.
type EmailOptions struct {
	NotifyAddresses []string `json:"notify-addresses" yaml:"notify-addresses" envconfig:"NOTIFY_ADDRESS"`
	DefaultFrom     string   `json:"default-from" yaml:"default-from" envconfig:"DEFAULT_FROM"`
	SubjectPrefix   string   `json:"subject-prefix" yaml:"subject-prefix" envconfig:"SUBJECT_PREFIX"`
	SMTPIdentity    string   `json:"smtp-identity" yaml:"smtp-identity" envconfig:"SMTP_IDENTITY"`
	SMTPUsername    string   `json:"smtp-username" yaml:"smtp-username" envconfig:"SMTP_USERNAME"`
	SMTPPassword    string   `json:"smtp-password" yaml:"smtp-password" envconfig:"SMTP_PASSWORD"`
	SMTPHost        string   `json:"smtp-host" yaml:"smtp-host" envconfig:"SMTP_HOST"`
	SMTPPort        int      `json:"smtp-port" yaml:"smtp-port" envconfig:"SMTP_PORT"`
	SMTPSkipVerify  bool     `json:"smtp-skip-verify" yaml:"smtp-skip-verify" envconfig:"SMTP_SKIP_VERIFY"`
}

// NewEmailOptions initializes EmailOptions with default values.
func NewEmailOptions() *EmailOptions {
	return &EmailOptions{
		NotifyAddresses: []string{},
		DefaultFrom:     Name + "@localhost",
		SubjectPrefix:   "[" + Name + "] ",
		SMTPIdentity:    "",
		SMTPUsername:    "",
		SMTPPassword:    "",
		SMTPHost:        "localhost",
		SMTPPort:        25,
		SMTPSkipVerify:  false,
	}
}

// Update updates options by loading email.json files.
func (o *EmailOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "email.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "email.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_email", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a JSON representation of the options.
func (o *EmailOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *EmailOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *EmailOptions) Prepare() error {
	return nil
}
