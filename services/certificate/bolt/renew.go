// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"net"
	"time"

	"github.com/boltdb/bolt"
)

// Renew requests a new SSL/TLS certificate before it expires.
func (s Service) Renew() error {
	return s.DB.View(func(tx *bolt.Tx) error {
		s.Logger.Debug("acme certificates renewal: started")
		defer s.Logger.Debug("acme certificates renewal: ended")

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

				addrs, err := net.LookupHost(fqdn)
				if err != nil {
					if e, ok := err.(*net.DNSError); ok {
						if e.Err == "no such host" {
							return
						}
					}
					s.Logger.Errorf("acme certificates renewal: lookup dns host: %s: %s", fqdn, err)
					return
				}

				if len(addrs) == 0 {
					s.Logger.Errorf("acme certificates renewal: lookup dns host: %s: no addresses returned", fqdn)
					return
				}

				cert, err := s.ObtainCertificate(fqdn)
				if err != nil {
					s.Logger.Errorf("acme certificates renewal: certificate renew: %s: %s", fqdn, err)
					return
				}
				s.Logger.Infof("acme certificates renewal: renewed: %s that expires %s", cert.FQDN, cert.ExpirationTime)
			}(string(k[keyTimeLayoutLen:]))
		}
		return nil
	})
}

// PeriodicRenew requests new SSL/TLS certificates on configured period.
func (s Service) PeriodicRenew() error {
	s.Logger.Info("acme certificates periodic renewal: initialized")
	go func() {
		defer s.RecoveryService.Recover()

		ticker := time.NewTicker(s.RenewCheckPeriod)
		defer ticker.Stop()
		if err := s.Renew(); err != nil {
			s.Logger.Errorf("acme certificates periodic renewal: %s", err)
		}
		for {
			select {
			case <-ticker.C:
				if err := s.Renew(); err != nil {
					s.Logger.Errorf("acme certificates periodic renewal: %s", err)
				}
			}
		}
	}()
	return nil
}
