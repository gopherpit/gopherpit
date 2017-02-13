// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package boltUser provides a Service that is using local BoltDB database
// to store User data.
package boltUser // import "gopherpit.com/gopherpit/services/user/bolt"

import (
	"os"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/user"
)

var (
	mmapFlags int
)

// Logger defines interface for logging messages with various severity levels.
type Logger interface {
	Info(a ...interface{})
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// Service implements gopherpit.com/gopherpit/services/user.Service interface.
type Service struct {
	DB *bolt.DB

	// PasswordNoReuseMonths is a number of months for which one password
	// can not be reused. If the value is 0 (default), reuse check is not
	// performed.
	PasswordNoReuseMonths int

	// If UsernameRequired is true, user.UsernameMissing will be returned
	// on CreateUser or UpdateUser if Username if empty string.
	UsernameRequired bool

	Logger Logger
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

// User retrieves a User instance by either ID, Email or Username as ref
// from a BoltDB database.
func (s Service) User(ref string) (u *user.User, err error) {
	var r *userRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecord(tx, []byte(ref))
		return
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// UserByID retrieves a User instance by ID from a BoltDB database.
func (s Service) UserByID(id string) (u *user.User, err error) {
	var r *userRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecordByID(tx, []byte(id))
		return
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// UserByEmail retrieves a User instance by email from a BoltDB database.
func (s Service) UserByEmail(email string) (u *user.User, err error) {
	var r *userRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecordByEmail(tx, []byte(email))
		return
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// UserByUsername retrieves a User instance by username from a BoltDB database.
func (s Service) UserByUsername(username string) (u *user.User, err error) {
	var r *userRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecordByUsername(tx, []byte(username))
		return
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// CreateUser creates a new User in a BoltDB database.
func (s Service) CreateUser(o *user.Options) (u *user.User, err error) {
	r := userRecord{}
	r.update(o)
	if err = s.DB.Update(func(tx *bolt.Tx) error {
		return r.save(tx, s.UsernameRequired)
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// UpdateUser changes data of an existing User.
func (s Service) UpdateUser(ref string, o *user.Options) (u *user.User, err error) {
	r := &userRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		r.update(o)
		return r.save(tx, s.UsernameRequired)
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// DeleteUser deletes an existing User.
func (s Service) DeleteUser(ref string) (u *user.User, err error) {
	r := &userRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		return r.delete(tx)
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// RegisterUser combines CreateUser SetPassword and RequestEmailChange
// into a single transaction to provide more convenient method
// for adding new users.
func (s Service) RegisterUser(o *user.Options, password string, emailValidationDeadline time.Time) (u *user.User, emailValidationToken string, err error) {
	r := userRecord{}
	r.update(o)
	if err = s.DB.Update(func(tx *bolt.Tx) error {
		if err := r.save(tx, s.UsernameRequired); err != nil {
			return err
		}
		salt, err := getUserSalt(tx, []byte(r.id))
		if err != nil {
			return err
		}
		if err := setPassword(tx, []byte(r.id), []byte(password), salt, s.PasswordNoReuseMonths); err != nil {
			return err
		}
		emailValidationToken, err = requestEmailChange(tx, r.id, r.Email, emailValidationDeadline)
		return err
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// SetPassword changes a password of an existing User.
func (s Service) SetPassword(ref string, password string) (err error) {
	return s.DB.Update(func(tx *bolt.Tx) error {
		id, err := getUserID(tx, []byte(ref))
		if err != nil {
			return err
		}
		salt, err := getUserSalt(tx, id)
		if err != nil {
			return err
		}
		return setPassword(tx, id, []byte(password), salt, s.PasswordNoReuseMonths)
	})
}

// RequestPasswordReset starts a process of reseting a password by
// providing a token that must be used in ResetPassword to authorize
// password reset.
func (s Service) RequestPasswordReset(ref string) (token string, err error) {
	err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		id, err := getUserID(tx, []byte(ref))
		if err != nil {
			return err
		}
		token, err = requestPasswordReset(tx, string(id))
		return
	})
	return
}

// ResetPassword changes a password of an existing User only if
// provided token is valid.
func (s Service) ResetPassword(token, password string) (err error) {
	return s.DB.Update(func(tx *bolt.Tx) (err error) {
		return resetPassword(tx, []byte(token), []byte(password), s.PasswordNoReuseMonths)
	})
}

// RequestEmailChange starts a process of changing an email by
// returning a token that must be used in ChangeEmail to authorize
// email change.
func (s Service) RequestEmailChange(ref, email string, validationDeadline time.Time) (token string, err error) {
	err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		id, err := getUserID(tx, []byte(ref))
		if err != nil {
			return err
		}
		token, err = requestEmailChange(tx, string(id), email, validationDeadline)
		return
	})
	return
}

// ChangeEmail changes an email of an existing User only if
// provided token is valid.
func (s Service) ChangeEmail(ref, token string) (u *user.User, err error) {
	r := &userRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getUserRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		if err = r.changeEmail(tx, token); err != nil {
			return
		}
		return r.save(tx, s.UsernameRequired)
	}); err != nil {
		return
	}
	u = r.export()
	return
}

// EmailChangeToken retrieves a token to change an email if it exists.
func (s Service) EmailChangeToken(ref, email string) (token string, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		id, err := getUserID(tx, []byte(ref))
		if err != nil {
			return err
		}
		token = emailChangeToken(tx, string(id), email)
		return
	})
	return
}

// UsersByID retrieves a paginated list of User instances ordered by
// ID values.
func (s Service) UsersByID(startID string, limit int) (page *user.UsersPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getUsersByID(tx, []byte(startID), limit)
		return
	})
	return
}

// UsersByEmail retrieves a paginated list of User instances ordered by
// Email values.
func (s Service) UsersByEmail(startEmail string, limit int) (page *user.UsersPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getUsersByEmail(tx, []byte(startEmail), limit)
		return
	})
	return
}

// UsersByUsername retrieves a paginated list of User instances ordered by
// Username values.
func (s Service) UsersByUsername(startUsername string, limit int) (page *user.UsersPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getUsersByUsername(tx, []byte(startUsername), limit)
		return
	})
	return
}

// Authenticate validates a password of an existing User.
func (s Service) Authenticate(ref, password string) (u *user.User, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err := getUserRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		salt, err := getUserSalt(tx, []byte(r.id))
		if err != nil {
			return
		}
		u = r.export()
		ok, err := authenticate(tx, []byte(r.id), []byte(password), salt)
		if !ok {
			return user.Unauthorized
		}
		return
	})
	return
}
