// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"resenje.org/logging"
	"resenje.org/web"

	"gopherpit.com/gopherpit/services/notification"
)

var emailRegex = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

func sendEmailValidationEmail(r *http.Request, to, token string) error {
	var textBody, htmlBody bytes.Buffer

	emailSettingsToken, err := tokenFromEmail(to)
	if err != nil {
		return fmt.Errorf("email settings token from email: %s", err)
	}

	if err := emailTemplateEmailValidateText.Execute(&textBody, map[string]interface{}{
		"Brand":              srv.Brand,
		"Host":               web.GetRequestEndpoint(r),
		"Token":              token,
		"EmailSettingsToken": string(emailSettingsToken),
	}); err != nil {
		return fmt.Errorf("emailTemplateEmailValidateText.Execute: %s", err)
	}

	if err := emailTemplateEmailValidateHTML.Execute(&htmlBody, map[string]interface{}{
		"Brand":              srv.Brand,
		"Host":               web.GetRequestEndpoint(r),
		"Token":              token,
		"EmailSettingsToken": string(emailSettingsToken),
	}); err != nil {
		return fmt.Errorf("emailTemplateEmailValidateHTML.Execute: %s", err)
	}
	id, err := srv.NotificationService.SendEmail(notification.Email{
		To:      []string{to},
		From:    srv.DefaultFrom,
		Subject: srv.Brand + " - E-mail address validation",
		Body:    textBody.String(),
		HTML:    htmlBody.String(),
	})
	if err != nil {
		return fmt.Errorf("notifier api send email: %s", err)
	}
	logging.Infof("email validation email sent to %s notifier id %s", to, id)
	return nil
}

func sendEmailPasswordResetEmail(r *http.Request, to, token string) error {
	var textBody, htmlBody bytes.Buffer

	emailSettingsToken, err := tokenFromEmail(to)
	if err != nil {
		return fmt.Errorf("email settings token from email: %s", err)
	}

	if err := emailTemplatePasswordResetText.Execute(&textBody, map[string]interface{}{
		"Brand":              srv.Brand,
		"Host":               web.GetRequestEndpoint(r),
		"Token":              token,
		"EmailSettingsToken": string(emailSettingsToken),
	}); err != nil {
		return fmt.Errorf("emailTemplatePasswordResetText.Execute: %s", err)
	}

	if err := emailTemplatePasswordResetHTML.Execute(&htmlBody, map[string]interface{}{
		"Brand":              srv.Brand,
		"Host":               web.GetRequestEndpoint(r),
		"Token":              token,
		"EmailSettingsToken": string(emailSettingsToken),
	}); err != nil {
		return fmt.Errorf("emailTemplatePasswordResetHTML.Execute: %s", err)
	}

	id, err := srv.NotificationService.SendEmail(notification.Email{
		To:      []string{to},
		From:    srv.DefaultFrom,
		Subject: srv.Brand + " - Password reset",
		Body:    textBody.String(),
		HTML:    htmlBody.String(),
	})
	if err != nil {
		return fmt.Errorf("notifier api send email: %s", err)
	}
	logging.Infof("password reset email sent to %s notifier id %s", to, id)
	return nil
}

func sendEmailContactEmail(replyTo, subject, message string) error {
	id, err := srv.NotificationService.SendEmail(notification.Email{
		To:      []string{srv.ContactRecipientEmail},
		From:    srv.DefaultFrom,
		ReplyTo: replyTo,
		Subject: subject,
		Body:    message,
	})
	if err != nil {
		return fmt.Errorf("notifier api send email: %s", err)
	}
	logging.Infof("contact email sent to %s notifier id %s", srv.ContactRecipientEmail, id)
	return nil
}

func tokenFromEmail(email string) ([]byte, error) {
	sum := md5.Sum(append(srv.salt, []byte(email)...))
	return encrypt(srv.salt[:16], append([]byte(email), sum[:]...))
}

func emailFromToken(token string) (string, error) {
	data, err := decrypt(srv.salt[:16], []byte(token))
	if err != nil {
		return "", err
	}
	signature := data[len(data)-md5.Size:]
	email := data[:len(data)-md5.Size]
	sum := md5.Sum(append(srv.salt, []byte(email)...))
	if !bytes.Equal(sum[:], signature) {
		return "", errors.New("invalid signature")
	}
	return string(email), nil
}
