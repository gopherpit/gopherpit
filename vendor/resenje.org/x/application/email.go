// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

// EmailOptions defines parameters for email sending.
type EmailOptions struct {
	name            string
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
func NewEmailOptions(name string) *EmailOptions {
	return &EmailOptions{
		name:            name,
		NotifyAddresses: []string{},
		DefaultFrom:     name + "@localhost",
		SubjectPrefix:   "[" + name + "] ",
		SMTPIdentity:    "",
		SMTPUsername:    "",
		SMTPPassword:    "",
		SMTPHost:        "localhost",
		SMTPPort:        25,
		SMTPSkipVerify:  false,
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *EmailOptions) VerifyAndPrepare() error { return nil }
