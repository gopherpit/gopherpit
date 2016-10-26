// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package certificateCache

import (
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"gopherpit.com/gopherpit/services/certificate"
)

// ErrCertificateNotFound is error that is returned if certificate in cache is nil.
var ErrCertificateNotFound = errors.New("certificate not found")

type tlsCertificate struct {
	*tls.Certificate
	TTL time.Time
}

// Cache represents a structure that holds cache map and certificate service.
type Cache struct {
	ttl               time.Duration
	ttlNoCert         time.Duration
	certificateGetter certificate.Getter
	nameToCertificate map[string]tlsCertificate
	mu                *sync.RWMutex
}

// NewCache creates a new instance of Cache.
func NewCache(certificateGetter certificate.Getter, ttl, ttlNoCert time.Duration) Cache {
	return Cache{
		ttl:               ttl,
		ttlNoCert:         ttlNoCert,
		certificateGetter: certificateGetter,
		nameToCertificate: map[string]tlsCertificate{},
		mu:                &sync.RWMutex{},
	}
}

// Certificate returns a tls.Certificate from cache or fetches it from
// certificate service.
func (cc Cache) Certificate(name string) (c *tls.Certificate, err error) {
	var found bool
	cc.mu.RLock()
	cert, found := cc.nameToCertificate[name]
	cc.mu.RUnlock()

	// Certificate found in cache
	if found {
		c = cert.Certificate
		switch {
		case time.Now().After(cert.TTL):
			// Is certificate cache expired
			cc.mu.Lock()
			delete(cc.nameToCertificate, name)
			cc.mu.Unlock()
		case c == nil:
			// Is certificate not found
			err = ErrCertificateNotFound
			return
		default:
			// All fine, return cached data
			return
		}
	}

	// Acquire certificate
	cc.mu.Lock()
	defer cc.mu.Unlock()

	crt, err := cc.certificateGetter.Certificate(name)
	if err != nil {
		// Set certificate cache to avoid frequent queries to pit api
		cc.nameToCertificate[name] = tlsCertificate{
			TTL: time.Now().Add(cc.ttlNoCert),
		}
		if err == certificate.CertificateNotFound {
			return
		}
		err = fmt.Errorf("certificate: %s: %s", name, err)
		return
	}
	if crt == nil {
		err = ErrCertificateNotFound
		return
	}
	c = &tls.Certificate{}
	*c, err = tls.X509KeyPair([]byte(crt.Cert), []byte(crt.Key))
	if err != nil {
		err = fmt.Errorf("tls X509KeyPair: %s: %s", name, err)
		return
	}

	cc.nameToCertificate[name] = tlsCertificate{
		Certificate: c,
		TTL:         time.Now().Add(cc.ttl),
	}
	return
}

// InvalidateCertificate removes certificate form cache for a given domain name
// if it exists.
func (cc Cache) InvalidateCertificate(name string) {
	cc.mu.Lock()
	delete(cc.nameToCertificate, name)
	cc.mu.Unlock()
}
