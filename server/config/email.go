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
)

// EmailOptions defines parameters for email sending.
type EmailOptions struct {
	NotifyAddresses []string `json:"notify-addresses" envconfig:"NOTIFY_ADDRESS"`
	DefaultFrom     string   `json:"default-from" envconfig:"DEFAULT_FROM"`
	SubjectPrefix   string   `json:"subject-prefix" envconfig:"SUBJECT_PREFIX"`
	SMTPIdentity    string   `json:"smtp-identity" envconfig:"SMTP_IDENTITY"`
	SMTPUsername    string   `json:"smtp-username" envconfig:"SMTP_USERNAME"`
	SMTPPassword    string   `json:"smtp-password" envconfig:"SMTP_PASSWORD"`
	SMTPHost        string   `json:"smtp-host" envconfig:"SMTP_HOST"`
	SMTPPort        int      `json:"smtp-port" envconfig:"SMTP_PORT"`
	SMTPSkipVerify  bool     `json:"smtp-skip-verify" envconfig:"SMTP_SKIP_VERIFY"`
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

// Update updates options by loading email.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *EmailOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "email.json")
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
	data, _ := json.MarshalIndent(o, "", "    ")
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
