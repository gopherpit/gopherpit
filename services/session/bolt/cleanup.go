// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltSession

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

type sessionCheck struct {
	Expires time.Time `json:"expires,omitempty"`
}

func cleanup(db *bolt.DB, logger Logger) {
	if err := db.View(func(tx *bolt.Tx) error {
		if sessionCount(tx) == 0 {
			return nil
		}
		logger.Debug("session cleanup: started")
		defer logger.Debug("session cleanup: ended")
		return tx.Bucket(bucketNameSessions).ForEach(func(k, v []byte) error {
			s := &sessionCheck{}
			if err := json.Unmarshal(v, s); err != nil {
				return err
			}
			if time.Now().After(s.Expires) {
				go func(key []byte) {
					if err := db.Batch(func(tx *bolt.Tx) error {
						return tx.Bucket(bucketNameSessions).Delete(key)
					}); err != nil {
						logger.Errorf("session cleanup: delete key %s:%s: %s", bucketNameSessions, key, err)
						return
					}
					logger.Debugf("session cleanup: deleted %s:%s", bucketNameSessions, key)
				}(k)
			}
			return nil
		})
	}); err != nil {
		logger.Errorf("session cleanup: for each key in %s: %s", bucketNameSessions, err)
	}
}

// PeriodicCleanup deletes expired session on a period defined in
// Service.CleanupPeriod.
func (s Service) PeriodicCleanup() (err error) {
	if s.CleanupPeriod <= 0 {
		s.Logger.Info("session cleanup: disabled")
		return
	}
	s.Logger.Info("session cleanup: initialized")
	go func() {
		defer func() {
			if err := recover(); err != nil {
				s.Logger.Errorf("session cleanup: panic: %s", err)
			}
		}()
		ticker := time.NewTicker(s.CleanupPeriod)
		defer ticker.Stop()
		cleanup(s.DB, s.Logger)
		for {
			select {
			case <-ticker.C:
				cleanup(s.DB, s.Logger)
			}
		}
	}()
	return
}
