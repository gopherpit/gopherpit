// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email // import "resenje.org/email"

import (
	"crypto/tls"

	"gopkg.in/gomail.v2"
)

// Service provides functionality to send emails over SMTP server.
type Service struct {
	// SMTP server host.
	SMTPHost string
	// SMTP server port.
	SMTPPort int
	// Do not verify SMTP hostname over encrypted connection.
	SMTPSkipVerify bool
	// SMTP identity.
	SMTPIdentity string
	// Username for SMTP server authentication.
	SMTPUsername string
	// Password for SMTP server authentication.
	SMTPPassword string
	// Adressess fot Notify method.
	NotifyAddresses []string
	// From address for Notify method.
	DefaultFrom string
	// Subject prefix for Notify method. It is not space separated from subject value.
	SubjectPrefix string
}

// SendEmail sends an email message.
func (s Service) SendEmail(from string, to []string, subject string, body string) error {
	return s.SendEmailWithHeaders(from, to, subject, body, nil)
}

// SendEmailWithHeaders sends an email message with additional headers.
func (s Service) SendEmailWithHeaders(from string, to []string, subject string, body string, headers map[string][]string) error {
	m := gomail.NewMessage()
	m.SetHeaders(headers)
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewPlainDialer(
		s.SMTPHost,
		s.SMTPPort,
		s.SMTPUsername,
		s.SMTPPassword,
	)
	d.LocalName = s.SMTPIdentity
	if s.SMTPSkipVerify {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return d.DialAndSend(m)
}

// Notify sends an email message to Service.NotifyAddresses.
func (s Service) Notify(subject, body string) error {
	return s.NotifyWithHeaders(subject, body, nil)
}

// NotifyWithHeaders sends an email message to Service.NotifyAddresses with additional headers.
func (s Service) NotifyWithHeaders(subject, body string, headers map[string][]string) error {
	if len(s.NotifyAddresses) == 0 {
		return nil
	}
	return s.SendEmailWithHeaders(s.DefaultFrom, s.NotifyAddresses, s.SubjectPrefix+subject, body, headers)
}
