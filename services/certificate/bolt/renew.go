// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"time"

	"github.com/boltdb/bolt"
	"resenje.org/logging"
)

// Renew requests a new SSL/TLS certificate before it expires.
func (s Service) Renew() error {
	var err error
	logger := s.Logger
	if logger == nil {
		logger, err = logging.GetLogger("default")
		if err != nil {
			panic(err)
		}
	}
	return s.DB.View(func(tx *bolt.Tx) error {
		logger.Debug("acme certificates renewal: started")
		defer logger.Debug("acme certificates renewal: ended")

		bucket := tx.Bucket(bucketNameIndexCertificateExpirationTimeFQDN)
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			et, err := time.Parse(keyTimeLayout, string(k[:keyTimeLayoutLen]))
			if err != nil {
				return err
			}
			if time.Now().Before(et.Add(-s.RenewPeriod)) {
				break
			}

			go func(fqdn string) {
				defer s.RecoveryService.Recover()

				cert, err := s.ObtainCertificate(fqdn)
				if err != nil {
					logger.Errorf("acme certificates renewal: certificate renew: %s: %s", fqdn, err)
					return
				}
				logger.Infof("acme certificates renewal: renewed: %s that expires %s", cert.FQDN, cert.ExpirationTime)
			}(string(k[keyTimeLayoutLen:]))
		}
		return nil
	})
}

// PeriodicRenew requests new SSL/TLS certificates on configured period.
func (s Service) PeriodicRenew() error {
	logger := s.Logger
	if logger == nil {
		var err error
		logger, err = logging.GetLogger("default")
		if err != nil {
			return err
		}
	}
	logger.Info("acme certificates periodic renewal: initialized")
	go func() {
		defer s.RecoveryService.Recover()

		ticker := time.NewTicker(s.RenewCheckPeriod)
		defer ticker.Stop()
		if err := s.Renew(); err != nil {
			logger.Errorf("acme certificates periodic renewal: %s", err)
		}
		for {
			select {
			case <-ticker.C:
				if err := s.Renew(); err != nil {
					logger.Errorf("acme certificates periodic renewal: %s", err)
				}
			}
		}
	}()
	return nil
}
