// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltUser

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/user"
)

const keyTimeFormat = "20060102150405"

var (
	bucketNameUsers              = []byte("Users")
	bucketNameUsersDeleted       = []byte("Users_Deleted")
	bucketNameIndexUsersUsername = []byte("Index_Users_Username")
	bucketNameIndexUsersEmail    = []byte("Index_Users_Email")
	bucketNameSalts              = []byte("Salts")
	bucketNamePasswords          = []byte("Passwords")
	bucketNameEmailValidations   = []byte("Email_Validations")
	bucketNamePasswordResets     = []byte("Password_Resets")
	emailRegex                   = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
	usernameRegex                = regexp.MustCompile(`^[\pL\pN]`)
)

type userRecord struct {
	id                    string
	Email                 string `json:"email,omitempty"`
	Username              string `json:"username,omitempty"`
	Name                  string `json:"name,omitempty"`
	Admin                 bool   `json:"admin,omitempty"`
	NotificationsDisabled bool   `json:"notifications-disabled,omitempty"`
	EmailUnvalidated      bool   `json:"email-unvalidated,omitempty"`
	Disabled              bool   `json:"disabled,omitempty"`
}

func (r userRecord) export() *user.User {
	return &user.User{
		ID:       r.id,
		Email:    r.Email,
		Username: r.Username,
		Name:     r.Name,
		Admin:    r.Admin,
		NotificationsDisabled: r.NotificationsDisabled,
		EmailUnvalidated:      r.EmailUnvalidated,
		Disabled:              r.Disabled,
	}
}

type passwordResetRecord struct {
	UserID         string    `json:"user-id"`
	ExpirationTime time.Time `json:"expiration-time"`
}

type emailValidationRecord struct {
	Email          string    `json:"email"`
	UserID         string    `json:"user-id"`
	ExpirationTime time.Time `json:"expiration-time"`
}

func getUserID(tx *bolt.Tx, ref []byte) (id []byte, err error) {
	ref = bytes.TrimSpace(ref)
	// Check if ref is email. It is possible to use regex as
	// username or ID can not have email format.
	if emailRegex.Match(ref) {
		bucket := tx.Bucket(bucketNameIndexUsersEmail)
		if bucket == nil {
			err = user.UserNotFound
			return
		}
		id = bucket.Get(bytes.ToLower(ref))
		if id == nil {
			err = user.UserNotFound
		}
		return
	}

	// Check if ref is username.
	bucket := tx.Bucket(bucketNameIndexUsersUsername)
	if bucket != nil {
		id = bucket.Get(ref)
		if id != nil {
			// The ID is found.
			return
		}
	}

	// Checking if ref is ID is acceptable as checks
	// to disallow username as existing ID and
	// to skip new IDs that match existing usernames.
	bucket = tx.Bucket(bucketNameUsers)
	if bucket == nil {
		err = user.UserNotFound
		return
	}
	if data := bucket.Get(ref); data == nil {
		err = user.UserNotFound
		return
	}
	id = ref
	return
}

func getUserSalt(tx *bolt.Tx, id []byte) (salt []byte, err error) {
	bucket := tx.Bucket(bucketNameSalts)
	if bucket == nil {
		err = user.SaltNotFound
		return
	}
	salt = bucket.Get(id)
	if salt == nil {
		err = user.SaltNotFound
		return
	}
	return
}

func saveUserSalt(tx *bolt.Tx, id, salt []byte) error {
	bucket, err := tx.CreateBucketIfNotExists(bucketNameSalts)
	if err != nil {
		return err
	}
	return bucket.Put(id, salt)
}

func getUserRecord(tx *bolt.Tx, ref []byte) (r *userRecord, err error) {
	if emailRegex.Match(ref) {
		return getUserRecordByEmail(tx, ref)
	}
	r, err = getUserRecordByUsername(tx, ref)
	if err != user.UserNotFound {
		return
	}
	return getUserRecordByID(tx, ref)
}

func getUserRecordByID(tx *bolt.Tx, id []byte) (r *userRecord, err error) {
	bucket := tx.Bucket(bucketNameUsers)
	if bucket == nil {
		err = user.UserNotFound
		return
	}
	data := bucket.Get(id)
	if data == nil {
		err = user.UserNotFound
		return
	}
	if err = json.Unmarshal(data, &r); err != nil {
		return
	}
	r.id = string(id)
	return
}

func getUserRecordByEmail(tx *bolt.Tx, email []byte) (u *userRecord, err error) {
	email = bytes.ToLower(email)
	bucket := tx.Bucket(bucketNameIndexUsersEmail)
	if bucket == nil {
		err = user.UserNotFound
		return
	}
	id := bucket.Get(email)
	if id == nil {
		err = user.UserNotFound
		return
	}
	return getUserRecordByID(tx, id)
}

func getUserRecordByUsername(tx *bolt.Tx, username []byte) (u *userRecord, err error) {
	username = bytes.TrimSpace(username)
	bucket := tx.Bucket(bucketNameIndexUsersUsername)
	if bucket == nil {
		err = user.UserNotFound
		return
	}
	id := bucket.Get(username)
	if id == nil {
		err = user.UserNotFound
		return
	}
	return getUserRecordByID(tx, id)
}

func emailExists(tx *bolt.Tx, email string) (exists bool, err error) {
	email = strings.ToLower(email)
	bucket := tx.Bucket(bucketNameIndexUsersEmail)
	if bucket != nil {
		if bucket.Get([]byte(email)) != nil {
			exists = true
			return
		}
	}
	bucket = tx.Bucket(bucketNameEmailValidations)
	if bucket != nil {
		err = bucket.ForEach(func(k, v []byte) error {
			emailValidation := emailValidationRecord{}
			if err := json.Unmarshal(v, &emailValidation); err != nil {
				if err := bucket.Delete(k); err != nil {
					return err
				}
				return err
			}
			if emailValidation.Email == email && time.Now().Before(emailValidation.ExpirationTime) {
				exists = true
			}
			return nil
		})
	}
	return
}

func (r *userRecord) update(o *user.Options) {
	if o == nil {
		return
	}
	if o.Email != nil {
		r.Email = *o.Email
	}
	if o.Username != nil {
		r.Username = *o.Username
	}
	if o.Name != nil {
		r.Name = *o.Name
	}
	if o.Admin != nil {
		r.Admin = *o.Admin
	}
	if o.NotificationsDisabled != nil {
		r.NotificationsDisabled = *o.NotificationsDisabled
	}
	if o.EmailUnvalidated != nil {
		r.EmailUnvalidated = *o.EmailUnvalidated
	}
	if o.Disabled != nil {
		r.Disabled = *o.Disabled
	}
}

func (r *userRecord) save(tx *bolt.Tx, usernameRequired bool) (err error) {
	// Required fields
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if r.Email == "" {
		return user.EmailMissing
	}
	if !emailRegex.MatchString(r.Email) {
		return user.EmailInvalid
	}
	r.Username = strings.TrimSpace(r.Username)
	if r.Username != "" {
		// Username can not be in email format.
		if emailRegex.MatchString(r.Username) {
			return user.UsernameInvalid
		}
		if !usernameRegex.MatchString(r.Username) {
			return user.UsernameInvalid
		}
		// Username can not be the same as existing ID.
		if bucket := tx.Bucket(bucketNameIndexUsersUsername); bucket != nil {
			data := bucket.Get([]byte(r.Username))
			if data != nil && string(data) != r.id {
				return user.UsernameExists
			}
		}
	} else if usernameRequired {
		return user.UsernameMissing
	}

	// existing user record
	re := &userRecord{}
	if r.id == "" {
		// Generate new id
		id, err := newUserID(tx)
		if err != nil {
			return fmt.Errorf("user record save generate unique ID: %s", err)
		}
		r.id = id
		if r.Email != "" {
			// Email must be unique
			exists, err := emailExists(tx, r.Email)
			if err != nil {
				return fmt.Errorf("user record save email exists: %s", err)
			}
			if exists {
				return user.EmailExists
			}
		}
	} else {
		// Check if user with u.id exists
		rc, err := getUserRecordByID(tx, []byte(r.id))
		if err != nil {
			return fmt.Errorf("user record save get user %s: %s", r.id, err)
		}
		if rc != nil {
			re = rc
		}
	}

	value, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("user record save json.Marshal %s", err)
	}
	byteID := []byte(r.id)

	// Username index
	if r.Username != re.Username {
		if re.Username != "" {
			bucket := tx.Bucket(bucketNameIndexUsersUsername)
			if bucket != nil {
				if err := bucket.Delete([]byte(re.Username)); err != nil {
					return fmt.Errorf("bucket(%s).Delete(%s) %s", bucketNameIndexUsersUsername, re.Username, err)
				}
			}
		}
		if r.Username != "" {
			bucket, err := tx.CreateBucketIfNotExists(bucketNameIndexUsersUsername)
			if err != nil {
				return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameIndexUsersUsername, err)
			}
			if err = bucket.Put([]byte(r.Username), byteID); err != nil {
				return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameIndexUsersUsername, r.Username, err)
			}
		}
	}

	// Email index
	if r.Email != re.Email {
		if re.Email != "" {
			bucket := tx.Bucket(bucketNameIndexUsersEmail)
			if bucket != nil {
				if err := bucket.Delete([]byte(re.Email)); err != nil {
					return fmt.Errorf("bucket(%s).Delete(%s) %s", bucketNameIndexUsersEmail, re.Email, err)
				}
			}
		}
		if r.Email != "" {
			// Save email index
			bucket, err := tx.CreateBucketIfNotExists(bucketNameIndexUsersEmail)
			if err != nil {
				return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameIndexUsersEmail, err)
			}
			if err = bucket.Put([]byte(r.Email), byteID); err != nil {
				return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameIndexUsersEmail, r.Email, err)
			}
		}
	}

	// Generate Salt if it is missing.
	if _, err := getUserSalt(tx, byteID); err == user.SaltNotFound {
		salt := make([]byte, 50)
		_, err := rand.Read(salt)
		if err != nil {
			return fmt.Errorf("generate new salt: %s", err)
		}
		if err = saveUserSalt(tx, byteID, salt); err != nil {
			return fmt.Errorf("save salt: %s", err)
		}
	}

	// Save user data
	bucket, err := tx.CreateBucketIfNotExists(bucketNameUsers)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameUsers, err)
	}
	if err := bucket.Put(byteID, value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameUsers, r.id, err)
	}

	return nil
}

func (r userRecord) delete(tx *bolt.Tx) (err error) {
	bucket, err := tx.CreateBucketIfNotExists(bucketNameUsers)
	if err != nil {
		return
	}
	id := []byte(r.id)
	if err = bucket.Delete(id); err != nil {
		return
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNameIndexUsersEmail)
	if err != nil {
		return
	}
	if err = bucket.Delete([]byte(r.Email)); err != nil {
		return
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNameIndexUsersUsername)
	if err != nil {
		return
	}
	if err = bucket.Delete([]byte(r.Username)); err != nil {
		return
	}
	value, err := json.Marshal(r)
	if err != nil {
		return
	}
	bucket = tx.Bucket(bucketNameEmailValidations)
	if bucket != nil {
		if err := bucket.ForEach(func(k, v []byte) error {
			emailValidation := emailValidationRecord{}
			if err := json.Unmarshal(v, &emailValidation); err != nil {
				// Ignore errors in this case
				return nil
			}
			if emailValidation.UserID == r.id {
				if err := bucket.Delete(k); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNameUsersDeleted)
	if err != nil {
		return
	}
	return bucket.Put(id, value)
}

func setPassword(tx *bolt.Tx, id, password, salt []byte, noReuseMonths int) (err error) {
	hash, err := passwordHash(password, salt)
	if err != nil {
		return
	}
	bucket, err := tx.CreateBucketIfNotExists(bucketNamePasswords)
	if err != nil {
		return err
	}
	bucket, err = bucket.CreateBucketIfNotExists(id)
	if err != nil {
		return err
	}
	if noReuseMonths > 0 {
		c := bucket.Cursor()
		for k, v := c.Seek([]byte(time.Now().UTC().AddDate(0, -noReuseMonths, 0).Format(keyTimeFormat))); k != nil; k, v = c.Next() {
			if bytes.Equal(v, hash) {
				return user.PasswordUsed
			}
		}
	}
	return bucket.Put([]byte(time.Now().UTC().Format(keyTimeFormat)), hash)
}

func authenticate(tx *bolt.Tx, id, password, salt []byte) (ok bool, err error) {
	hash, err := passwordHash(password, salt)
	if err != nil {
		return
	}
	bucket := tx.Bucket(bucketNamePasswords)
	if bucket == nil {
		// No passwords in the system
		return
	}
	bucket = bucket.Bucket(id)
	if bucket == nil {
		// No passwords for this user
		return
	}
	_, v := bucket.Cursor().Last()
	if len(v) > 0 && bytes.Equal(v, hash) {
		ok = true
	}
	return
}

func requestPasswordReset(tx *bolt.Tx, id string) (token string, err error) {
	var v []byte
	v, err = json.Marshal(passwordResetRecord{
		UserID:         id,
		ExpirationTime: time.Now().AddDate(0, 0, 1),
	})
	if err != nil {
		return
	}
	token, err = newPasswordResetToken(tx)
	if err != nil {
		return
	}
	bucket, err := tx.CreateBucketIfNotExists(bucketNamePasswordResets)
	if err != nil {
		return
	}
	return token, bucket.Put([]byte(token), v)
}

func resetPassword(tx *bolt.Tx, token, password []byte, noReuseMonths int) (err error) {
	var bucket *bolt.Bucket
	bucket, err = tx.CreateBucketIfNotExists(bucketNamePasswordResets)
	if err != nil {
		return
	}
	data := bucket.Get(token)
	if len(data) == 0 {
		return user.PasswordResetTokenNotFound
	}
	v := passwordResetRecord{}
	if err = json.Unmarshal(data, &v); err != nil {
		return
	}
	if time.Now().After(v.ExpirationTime) {
		return user.PasswordResetTokenExpired
	}
	salt, err := getUserSalt(tx, []byte(v.UserID))
	if err != nil {
		return
	}
	if err = setPassword(tx, []byte(v.UserID), password, salt, noReuseMonths); err != nil {
		return
	}
	if err = bucket.Delete([]byte(token)); err != nil {
		return
	}
	return
}

func requestEmailChange(tx *bolt.Tx, id string, email string, validationExpirationTime time.Time) (token string, err error) {
	email = strings.ToLower(email)
	var r *userRecord
	r, err = getUserRecordByEmail(tx, []byte(email))
	switch err {
	case nil:
		if r != nil && r.id != id {
			err = user.EmailChangeEmailNotAvaliable
			return
		}
	case user.UserNotFound:
		err = nil
	default:
		return
	}
	var bucket *bolt.Bucket
	bucket, err = tx.CreateBucketIfNotExists(bucketNameEmailValidations)
	if err != nil {
		return token, fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameEmailValidations, err)
	}
	if err = bucket.ForEach(func(k, v []byte) error {
		emailValidation := emailValidationRecord{}
		if err := json.Unmarshal(v, &emailValidation); err != nil {
			return nil
		}
		if emailValidation.Email == email {
			if emailValidation.UserID == id {
				if err := bucket.Delete(k); err != nil {
					return fmt.Errorf("bucket(%s).Delete(%s) %s", bucketNameEmailValidations, k, err)
				}
			} else {
				return user.EmailChangeEmailNotAvaliable
			}
		}
		return nil
	}); err != nil {
		return
	}
	var v []byte
	v, err = json.Marshal(emailValidationRecord{
		Email:          email,
		UserID:         id,
		ExpirationTime: validationExpirationTime,
	})
	if err != nil {
		return token, fmt.Errorf("json.Marshal %s", err)
	}
	token, err = newEmailChangeToken(tx)
	if err != nil {
		return token, fmt.Errorf("new email change token: %s", err)
	}
	if err = bucket.Put([]byte(token), v); err != nil {
		return token, fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameEmailValidations, token, err)
	}
	return
}

func emailChangeToken(tx *bolt.Tx, id, email string) (token string) {
	bucket := tx.Bucket(bucketNameEmailValidations)
	if bucket == nil {
		return
	}
	c := bucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		emailValidation := emailValidationRecord{}
		if err := json.Unmarshal(v, &emailValidation); err != nil {
			continue
		}
		if emailValidation.Email == email && emailValidation.UserID == id {
			token = string(k)
			return
		}
	}
	return
}

func (r *userRecord) changeEmail(tx *bolt.Tx, token string) (err error) {
	var bucket *bolt.Bucket
	bucket, err = tx.CreateBucketIfNotExists(bucketNameEmailValidations)
	if err != nil {
		return
	}
	data := bucket.Get([]byte(token))
	if len(data) == 0 {
		return user.EmailValidateTokenNotFound
	}
	v := emailValidationRecord{}
	if err = json.Unmarshal(data, &v); err != nil {
		return
	}
	if r.id != v.UserID {
		return user.EmailValidateTokenInvalid
	}
	if !v.ExpirationTime.IsZero() && time.Now().After(v.ExpirationTime) {
		return user.EmailValidateTokenExpired
	}
	if err = bucket.Delete([]byte(token)); err != nil {
		return
	}
	r.Email = strings.ToLower(v.Email)
	r.EmailUnvalidated = false
	return nil
}

func getUsersByID(tx *bolt.Tx, start []byte, limit int) (page *user.UsersPage, err error) {
	bucket := tx.Bucket(bucketNameUsers)
	if bucket != nil {
		return
	}
	c := bucket.Cursor()
	var k, v []byte
	if len(start) == 0 {
		k, v = c.First()
	} else {
		k, v = c.Seek(start)
		var prev, p []byte
		for i := 0; i < limit; i++ {
			p, _ = c.Prev()
			if p == nil {
				break
			}
			prev = p
		}
		page.Previous = string(prev)
		k, v = c.Seek(start)
	}
	var i int
	var r *userRecord
	for i = 0; k != nil && i < limit; i++ {
		r = &userRecord{}
		if err = json.Unmarshal(v, &r); err != nil {
			return
		}
		r.id = string(k)
		u := r.export()
		page.Users = append(page.Users, *u)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}

func getUsersByEmail(tx *bolt.Tx, start []byte, limit int) (page *user.UsersPage, err error) {
	bucket := tx.Bucket(bucketNameIndexUsersEmail)
	if bucket == nil {
		return
	}
	c := bucket.Cursor()
	var k, v []byte
	if len(start) == 0 {
		k, v = c.First()
	} else {
		k, v = c.Seek(start)
		var prev, p []byte
		for i := 0; i < limit; i++ {
			p, _ = c.Prev()
			if p == nil {
				break
			}
			prev = p
		}
		page.Previous = string(prev)
		k, v = c.Seek(start)
	}
	var i int
	var r *userRecord
	for i = 0; k != nil && i < limit; i++ {
		r, err = getUserRecord(tx, v)
		if err != nil {
			return
		}
		u := r.export()
		page.Users = append(page.Users, *u)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}

func getUsersByUsername(tx *bolt.Tx, start []byte, limit int) (page *user.UsersPage, err error) {
	bucket := tx.Bucket(bucketNameIndexUsersUsername)
	if bucket == nil {
		return
	}
	c := bucket.Cursor()
	var k, v []byte
	if len(start) == 0 {
		k, v = c.First()
	} else {
		k, v = c.Seek(start)
		var prev, p []byte
		for i := 0; i < limit; i++ {
			p, _ = c.Prev()
			if p == nil {
				break
			}
			prev = p
		}
		page.Previous = string(prev)
		k, v = c.Seek(start)
	}
	var i int
	var r *userRecord
	for i = 0; k != nil && i < limit; i++ {
		r, err = getUserRecord(tx, v)
		if err != nil {
			return
		}
		u := r.export()
		page.Users = append(page.Users, *u)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}

var base32Encoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz")

func newUserID(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 10)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNameUsers)
	if err != nil {
		return
	}
	bucketUsernames, err := tx.CreateBucketIfNotExists(bucketNameIndexUsersUsername)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		_, err = rand.Read(br)
		if err != nil {
			return
		}
		b := append(bt, append(bp, br...)...)
		id = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		// Check if there already is an ID with the same value.
		if v := bucket.Get([]byte(id)); v == nil {
			// Check if there is a Username with the same value.
			// This check is required to be able to identify user
			// by either ID or Username.
			if v = bucketUsernames.Get([]byte(id)); v == nil {
				return
			}
		}
	}
	return "", errors.New("unable to generate unique user id")
}

func newPasswordResetToken(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 19)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNamePasswordResets)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		_, err = rand.Read(br)
		if err != nil {
			return
		}
		b := append(bt, append(bp, br...)...)
		id = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		if v := bucket.Get([]byte(id)); v == nil {
			return
		}
	}
	return "", errors.New("unable to generate unique password reset token")
}

func newEmailChangeToken(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 19)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNameEmailValidations)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		_, err = rand.Read(br)
		if err != nil {
			return
		}
		b := append(bt, append(bp, br...)...)
		id = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		if v := bucket.Get([]byte(id)); v == nil {
			return
		}
	}
	return "", errors.New("unable to generate unique email change token")
}

func passwordHash(password, salt []byte) ([]byte, error) {
	h, err := scrypt.Key(password, salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	return append([]byte("{scrypt}"), h...), nil
}
