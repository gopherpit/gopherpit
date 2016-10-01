// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notification // import "gopherpit.com/gopherpit/services/notification"

// Email represents an e-mail message.
type Email struct {
	From      string   `json:"from"`
	To        []string `json:"to"`
	CC        []string `json:"cc"`
	BCC       []string `json:"bcc"`
	ReplyTo   string   `json:"reply-to"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	HTML      string   `json:"html"`
	CheckSent bool     `json:"check-sent"`
}

// Service defines functions that Notification Service must implement.
type Service interface {
	// SendEmail sends an e-mail message and retuns it's ID.
	SendEmail(email Email) (id string, err error)
	// IsEmailOptedOut returns true or false if e-mail address
	// is marked not to send any e-mail messages to.
	IsEmailOptedOut(email string) (yes bool, err error)
	// OptOutEmail marks an e-mail address not to send any e-mail messages to.
	OptOutEmail(email string) (err error)
	// RemoveOptedOutEmail removes an opt-out mark previosulu set by
	// OptOutEmail.
	RemoveOptedOutEmail(email string) (err error)
}

// Errors that are related to the Notification Service.
var (
	EmailAlreadySent = NewError(1000, "email already sent")
)
