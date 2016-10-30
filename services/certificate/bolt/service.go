// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"crypto/x509"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/xenolf/lego/acme"
	"resenje.org/logging"
	"resenje.org/recovery"

	"gopherpit.com/gopherpit/services/certificate"
)

var (
	mmapFlags  int
	emailRegex = regexp.MustCompile("^[^@]+@[^@]+\\.[^@]+$")
)

// Service implements gopherpit.com/gopherpit/services/certificate.Service interface.
type Service struct {
	DB *bolt.DB

	// Default ACME directory URL.
	DefaultACMEDirectoryURL string

	// RenewPeriod is a duration after issuing a certificate
	// to try to renew it.
	RenewPeriod time.Duration
	// RenewCheckPeriod is a period of renewal process.
	RenewCheckPeriod time.Duration

	// RecoveryService recovers from panics, and logs and informs about it.
	RecoveryService recovery.Service
	// Default logger for this service.
	Logger *logging.Logger

	acmeUserCache *acmeUser
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

// Certificate returns a Certificate for provided FQDN.
func (s Service) Certificate(fqdn string) (c *certificate.Certificate, err error) {
	var r *certificateRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getCertificateRecord(tx, []byte(fqdn))
		return
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// ObtainCertificate requests a new SSL/TLS certificate from
// ACME provider and returns an instance of Certificate.
func (s Service) ObtainCertificate(fqdn string) (c *certificate.Certificate, err error) {
	mu.Lock()
	fqdnLock[fqdn] = struct{}{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		delete(fqdnLock, fqdn)
		mu.Unlock()
	}()

	var u *acmeUser
	u, err = s.acmeUser()
	if err != nil {
		return
	}
	var client *acme.Client
	client, err = acme.NewClient(u.DirectoryURL, u, acme.RSA2048)
	if err != nil {
		return
	}
	if err = client.SetChallengeProvider(acme.HTTP01, challengeProvider{s}); err != nil {
		return
	}
	client.ExcludeChallenges([]acme.Challenge{acme.TLSSNI01, acme.DNS01})

	certResource, failures := client.ObtainCertificate([]string{fqdn}, true, nil, false)
	if len(failures) > 0 {
		err = failures[fqdn]
		return
	}

	cert := string(certResource.Certificate)
	key := string(certResource.PrivateKey)
	return s.UpdateCertificate(certResource.Domain, &certificate.Options{
		Cert:          &cert,
		Key:           &key,
		ACMEURL:       &certResource.CertURL,
		ACMEURLStable: &certResource.CertStableURL,
		ACMEAccount:   &certResource.AccountRef,
	})
}

// IsCertificateBeingObtained tests if certificate is being obtained currently.
func (s Service) IsCertificateBeingObtained(fqdn string) (yes bool, err error) {
	mu.RLock()
	_, yes = fqdnLock[fqdn]
	mu.RUnlock()
	return
}

// UpdateCertificate alters the fields of existing Certificate.
func (s Service) UpdateCertificate(fqdn string, o *certificate.Options) (c *certificate.Certificate, err error) {
	logger := s.Logger
	if logger == nil {
		logger, err = logging.GetLogger("default")
		if err != nil {
			return
		}
	}
	var expirationTime *time.Time
	if o.Cert != nil {
		certs, err := loadPEMCertificates([]byte(*o.Cert))
		if err != nil {
			return nil, err
		}
	Loop:
		for _, cert := range certs {
			c, err := x509.ParseCertificate(cert)
			if err != nil {
				logger.Warningf("update certificate: %s: x509 parse certificate: %s", fqdn, err)
				return nil, certificate.CertificateInvalid
			}
			for _, name := range c.DNSNames {
				if strings.HasPrefix(name, "*.") {
					index := strings.Index(fqdn, ".")
					if index <= 0 {
						continue
					}
					name = fmt.Sprintf("%s.%s", fqdn[:index], name[2:])
				}
				if name == fqdn {
					expirationTime = &c.NotAfter
					break Loop
				}
			}
		}
		if expirationTime == nil {
			logger.Warningf("update certificate: %s: expiration time not found for dns name", fqdn)
			return nil, certificate.CertificateInvalid
		}
	}

	r := &certificateRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getCertificateRecord(tx, []byte(fqdn))
		switch err {
		case certificate.CertificateNotFound:
			r = &certificateRecord{
				fqdn: fqdn,
			}
		case nil:
		default:
			return
		}
		r.update(o, expirationTime)
		return r.save(tx)
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// DeleteCertificate deletes an existing Certificate for a
// provided FQDN and returns it.
func (s Service) DeleteCertificate(fqdn string) (c *certificate.Certificate, err error) {
	r := &certificateRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getCertificateRecord(tx, []byte(fqdn))
		if err != nil {
			return
		}
		return r.delete(tx)
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// Certificates retrieves a paginated list of Certificate instances
// ordered by FQDN.
func (s Service) Certificates(start string, limit int) (page *certificate.CertificatesPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getCertificates(tx, []byte(start), limit)
		return
	})
	return
}

// CertificatesInfoByExpiry retrieves a paginated list of Info instances
// ordered by expiration time.
func (s Service) CertificatesInfoByExpiry(since time.Time, start string, limit int) (page *certificate.InfosPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getCertificatesByExpiry(tx, since, []byte(start), limit)
		return
	})
	return
}

// ACMEUser returns ACME user with ACME authentication details.
func (s Service) ACMEUser() (u *certificate.ACMEUser, err error) {
	var au *acmeUser
	au, err = s.acmeUser()
	if err != nil {
		return
	}
	u = au.export().export()
	return
}

// RegisterACMEUser registers and saves ACME user authentication data.
func (s Service) RegisterACMEUser(directoryURL, email string) (u *certificate.ACMEUser, err error) {
	if email != "" {
		if !emailRegex.MatchString(email) {
			err = certificate.ACMEUserEmailInvalid
			return
		}
	}
	au, err := registerACMEUser(directoryURL, email, s.acmeUserFilename())
	if err != nil {
		return
	}
	u = au.export().export()
	return
}

// ACMEChallenge returns an instance of ACMEChallenge for a FQDN.
func (s Service) ACMEChallenge(fqdn string) (c *certificate.ACMEChallenge, err error) {
	var r *acmeChallengeRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getACMEChallengeRecord(tx, []byte(fqdn))
		return
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// UpdateACMEChallenge alters the fields of existing ACMEChallenge.
func (s Service) UpdateACMEChallenge(fqdn string, o *certificate.ACMEChallengeOptions) (c *certificate.ACMEChallenge, err error) {
	r := &acmeChallengeRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getACMEChallengeRecord(tx, []byte(fqdn))
		switch err {
		case certificate.ACMEChallengeNotFound:
			r = &acmeChallengeRecord{
				fqdn: fqdn,
			}
		case nil:
		default:
			return
		}
		r.update(o)
		return r.save(tx)
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// DeleteACMEChallenge deletes an existing ACMEChallenge for a
// provided FQDN and returns it.
func (s Service) DeleteACMEChallenge(fqdn string) (c *certificate.ACMEChallenge, err error) {
	r := &acmeChallengeRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getACMEChallengeRecord(tx, []byte(fqdn))
		if err != nil {
			return
		}
		return r.delete(tx)
	}); err != nil {
		return
	}
	c = r.export()
	return
}

// ACMEChallenges retrieves a paginated list of ACMEChallenge instances.
func (s Service) ACMEChallenges(start string, limit int) (page *certificate.ACMEChallengesPage, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		page, err = getACMEChallenges(tx, []byte(start), limit)
		return
	})
	return
}
