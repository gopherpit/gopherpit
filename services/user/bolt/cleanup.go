// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltUser

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/user"
)

// PeriodicCleanup deletes expired password resets and email validations
// periodically.
func (s Service) PeriodicCleanup() (err error) {
	s.Logger.Info("user password resets cleanup: initialized")
	s.Logger.Info("user email validations cleanup: initialized")
	go func() {
		for {
			// Clean email validations
			go func() {
				defer func() {
					if err := recover(); err != nil {
						s.Logger.Errorf("user email validations cleanup: panic: %s", err)
					}
				}()
				now := time.Now()
				if err := s.DB.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(bucketNameEmailValidations)
					if bucket == nil {
						return nil
					}
					return bucket.ForEach(func(k, v []byte) error {
						emailValidation := emailValidationRecord{}
						if err := json.Unmarshal(v, &emailValidation); err != nil {
							if err := bucket.Delete(k); err != nil {
								return err
							}
							return err
						}
						if emailValidation.ExpirationTime.IsZero() {
							if _, err := getUserRecordByID(tx, []byte(emailValidation.UserID)); err == user.UserNotFound {
								if err := bucket.Delete(k); err != nil {
									return err
								}
								s.Logger.Infof("user email validations cleanup: deleted validation for user id %s email %s as user is not found", emailValidation.UserID, emailValidation.Email)
							}
							return nil
						}
						if now.After(emailValidation.ExpirationTime) {
							if err := bucket.Delete(k); err != nil {
								return err
							}
							s.Logger.Infof("user email validations cleanup: deleted validation for user id %s email %s", emailValidation.UserID, emailValidation.Email)
						}
						return nil
					})
				}); err != nil {
					s.Logger.Errorf("user email validations cleanup: %s", err)
				}
			}()
			// Clean password resets
			go func() {
				defer func() {
					if err := recover(); err != nil {
						s.Logger.Errorf("user password resets cleanup: panic: %s", err)
					}
				}()
				now := time.Now()
				if err := s.DB.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(bucketNamePasswordResets)
					if bucket == nil {
						return nil
					}
					return bucket.ForEach(func(k, v []byte) error {
						passwordReset := passwordResetRecord{}
						if err := json.Unmarshal(v, &passwordReset); err != nil {
							if err := bucket.Delete(k); err != nil {
								return err
							}
							return err
						}
						if now.After(passwordReset.ExpirationTime) {
							if err := bucket.Delete(k); err != nil {
								return err
							}
							s.Logger.Infof("clean passwords reset: deleted password reset for user id %s", passwordReset.UserID)
						}
						return nil
					})
				}); err != nil {
					s.Logger.Errorf("user password resets cleanup: %s", err)
				}
			}()
			time.Sleep(3 * time.Hour)
		}
	}()
	return
}
