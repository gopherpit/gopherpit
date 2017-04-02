// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltNotification // import "gopherpit.com/gopherpit/services/notification/bolt"

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/sha3"
	"gopkg.in/gomail.v2"

	"gopherpit.com/gopherpit/services/notification"
)

var (
	mmapFlags int
)

var (
	bucketNameEmailAddressOptOut = []byte("EmailAddressOptOut")
	bucketNameMessageIDs         = []byte("MessageIDs")

	keyTimeFormat = "20060102150405"
)

// Logger defines interface for logging messages with various severity levels.
type Logger interface {
	Info(a ...interface{})
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// Service implements gopherpit.com/gopherpit/services/notification.Service interface.
type Service struct {
	DB *bolt.DB

	SMTPHost       string
	SMTPPort       int
	SMTPUsername   string
	SMTPPassword   string
	SMTPIdentity   string
	SMTPSkipVerify bool
	CleanupPeriod  time.Duration
	Logger         Logger

	dialer *gomail.Dialer
}

// NewDB opens a new BoltDB database.
func NewDB(filename string, fileMode os.FileMode, boltOptions *bolt.Options) (db *bolt.DB, err error) {
	if boltOptions == nil {
		boltOptions = &bolt.Options{
			Timeout:   2 * time.Second,
			MmapFlags: mmapFlags,
		}
	}
	if fileMode == 0 {
		fileMode = 0640
	}
	db, err = bolt.Open(filename, fileMode, boltOptions)
	return
}

// SendEmail sends an e-mail message and returns it's ID.
func (s Service) SendEmail(email notification.Email) (id string, err error) {
	idb := make([]byte, 10)
	now := time.Now()
	sha3.ShakeSum256(idb, []byte(strings.Join(append(email.To, []string{email.From, email.Subject, email.Body, email.HTML, now.String()}...), "")))
	id = fmt.Sprintf("%x", idb)

	if email.CheckSent {
		err = s.DB.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists(bucketNameMessageIDs)
			if err != nil {
				return err
			}
			if e := bucket.Get(idb); e != nil {
				return notification.ErrEmailAlreadySent
			}
			return bucket.Put(idb, []byte(now.AddDate(0, 0, 21).Format(time.RFC3339)))
		})
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				if derr := s.DB.Update(func(tx *bolt.Tx) error {
					bucket, err := tx.CreateBucketIfNotExists(bucketNameMessageIDs)
					if err != nil {
						return err
					}
					return bucket.Delete(idb)
				}); derr != nil && s.Logger != nil {
					s.Logger.Errorf("send email: deleting email id %s: %s", id, derr)
				}
			}
		}()
	}

	message := gomail.NewMessage()
	message.SetHeader("From", email.From)
	message.SetHeader("To", email.To...)
	if email.CC != nil {
		message.SetHeader("CC", email.CC...)
	}
	if email.BCC != nil {
		message.SetHeader("BCC", email.BCC...)
	}
	if email.ReplyTo != "" {
		message.SetHeader("Reply-To", email.ReplyTo)
	}
	message.SetHeader("Subject", email.Subject)
	message.SetHeader("X-ID", id)
	message.SetDateHeader("X-Date", now)
	if email.Body != "" {
		message.SetBody("text/plain", email.Body)
		if email.HTML != "" {
			message.AddAlternative("text/html", email.HTML)
		}
	} else {
		message.SetBody("text/html", email.HTML)
	}
	err = s.getDialer().DialAndSend(message)
	return
}

// IsEmailOptedOut returns true or false if e-mail address is marked not to
// send any e-mail messages to.
func (s Service) IsEmailOptedOut(email string) (yes bool, err error) {
	email = strings.TrimSpace(strings.ToLower(email))
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(bucketNameEmailAddressOptOut)
		if bucket == nil {
			return
		}
		data := bucket.Get([]byte(email))
		if data != nil {
			yes = true
			return
		}
		return
	})
	return
}

// OptOutEmail marks an e-mail address not to send any e-mail messages to.
func (s Service) OptOutEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	return s.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketNameEmailAddressOptOut)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(email), []byte(time.Now().UTC().Format(keyTimeFormat)))
	})
}

// RemoveOptedOutEmail removes an opt-out mark previosulu set by OptOutEmail.
func (s Service) RemoveOptedOutEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	return s.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketNameEmailAddressOptOut)
		if err != nil {
			return err
		}
		return bucket.Delete([]byte(email))
	})
}

func (s Service) getDialer() *gomail.Dialer {
	if s.dialer == nil {
		s.dialer = gomail.NewPlainDialer(
			s.SMTPHost,
			s.SMTPPort,
			s.SMTPUsername,
			s.SMTPPassword,
		)
		s.dialer.LocalName = s.SMTPIdentity
		if s.SMTPSkipVerify {
			s.dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}
	return s.dialer
}
