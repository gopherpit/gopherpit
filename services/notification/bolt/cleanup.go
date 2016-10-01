// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltNotification

import (
	"time"

	"github.com/boltdb/bolt"
	"resenje.org/logging"
)

type sessionCheck struct {
	Expires time.Time `json:"expires,omitempty"`
}

func cleanup(db *bolt.DB, logger *logging.Logger) {
	now := time.Now()
	if err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketNameMessageIDs)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			expire, err := time.Parse(time.RFC3339, string(v))
			if err != nil {
				return err
			}
			if now.After(expire) {
				if err := bucket.Delete(k); err != nil {
					return err
				}
				logging.Infof("expired email message cleanup: deleted id %x", k)
			}
			return nil
		})
	}); err != nil {
		logging.Errorf("expired email message cleanup: %s", err)
	}
}

// PeriodicCleanup deletes expired email message IDs on a period
// defined in Service.CleanupPeriod.
func (s Service) PeriodicCleanup(logger *logging.Logger) (err error) {
	if logger == nil {
		logger, err = logging.GetLogger("default")
		if err != nil {
			return
		}
	}
	if s.CleanupPeriod <= 0 {
		logger.Info("expired email message cleanup: disabled")
		return
	}
	logger.Info("expired email message cleanup: initialized")
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("expired email message cleanup: panic: %s", err)
			}
		}()
		ticker := time.NewTicker(s.CleanupPeriod)
		defer ticker.Stop()
		cleanup(s.DB, logger)
		for {
			select {
			case <-ticker.C:
				cleanup(s.DB, logger)
			}
		}
	}()
	return
}
