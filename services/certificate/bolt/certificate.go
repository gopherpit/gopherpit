// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/pkg/boltutils"
	"gopherpit.com/gopherpit/services/certificate"
)

var (
	bucketNameCertificates                       = []byte("Certificates")
	bucketNameIndexCertificateExpirationTimeFQDN = []byte("Index_CertificateExpirationTime_FQDN")

	keyTimeLayout    = "20060102150405.000000000"
	keyTimeLayoutLen = len(keyTimeLayout)

	fqdnLock = map[string]struct{}{}
	mu       = &sync.RWMutex{}
)

type certificateRecord struct {
	fqdn           string
	ExpirationTime *time.Time `json:"expiration-time,omitempty"`
	Cert           string     `json:"cert,omitempty"`
	Key            string     `json:"key,omitempty"`
	ACMEURL        string     `json:"acme-url,omitempty"`
	ACMEURLStable  string     `json:"acme-url-stable,omitempty"`
	ACMEAccount    string     `json:"acme-account,omitempty"`
}

func (t certificateRecord) export() *certificate.Certificate {
	return &certificate.Certificate{
		FQDN:           t.fqdn,
		ExpirationTime: t.ExpirationTime,
		Cert:           t.Cert,
		Key:            t.Key,
		ACMEURL:        t.ACMEURL,
		ACMEURLStable:  t.ACMEURLStable,
		ACMEAccount:    t.ACMEAccount,
	}
}

func (t *certificateRecord) update(o *certificate.Options, expirationTime *time.Time) error {
	if expirationTime != nil {
		t.ExpirationTime = expirationTime
	}
	if o == nil {
		return nil
	}
	if o.Cert != nil {
		t.Cert = *o.Cert
	}
	if o.Key != nil {
		t.Key = *o.Key
	}
	if o.ACMEURL != nil {
		t.ACMEURL = *o.ACMEURL
	}
	if o.ACMEURLStable != nil {
		t.ACMEURLStable = *o.ACMEURLStable
	}
	if o.ACMEURL != nil {
		t.ACMEAccount = *o.ACMEAccount
	}
	return nil
}

func getCertificateRecord(tx *bolt.Tx, fqdn []byte) (t *certificateRecord, err error) {
	bucket := tx.Bucket(bucketNameCertificates)
	if bucket == nil {
		err = certificate.CertificateNotFound
		return
	}
	data := bucket.Get(fqdn)
	if data == nil {
		err = certificate.CertificateNotFound
		return
	}
	if err = json.Unmarshal(data, &t); err != nil {
		return
	}
	t.fqdn = string(fqdn)
	return
}

type certificateInfo struct {
	fqdn           string
	ExpirationTime *time.Time `json:"expiration-time,omitempty"`
	ACMEURL        string     `json:"acme-url,omitempty"`
	ACMEURLStable  string     `json:"acme-url-stable,omitempty"`
	ACMEAccount    string     `json:"acme-account,omitempty"`
}

func (t certificateInfo) export() *certificate.Info {
	return &certificate.Info{
		FQDN:           t.fqdn,
		ExpirationTime: t.ExpirationTime,
		ACMEURL:        t.ACMEURL,
		ACMEURLStable:  t.ACMEURLStable,
		ACMEAccount:    t.ACMEAccount,
	}
}

func getCertificateInfo(tx *bolt.Tx, fqdn []byte) (t *certificateInfo, err error) {
	bucket := tx.Bucket(bucketNameCertificates)
	if bucket == nil {
		err = certificate.CertificateNotFound
		return
	}
	data := bucket.Get(fqdn)
	if data == nil {
		err = certificate.CertificateNotFound
		return
	}
	if err = json.Unmarshal(data, &t); err != nil {
		return
	}
	t.fqdn = string(fqdn)
	return
}

func (t *certificateRecord) save(tx *bolt.Tx) (err error) {
	t.fqdn = strings.ToLower(strings.TrimSpace(t.fqdn))
	// Required fields
	if t.fqdn == "" {
		return certificate.FQDNMissing
	}

	// existing certificate record
	et := certificateRecord{}
	ct, err := getCertificateRecord(tx, []byte(t.fqdn))
	if err != nil && err != certificate.CertificateNotFound {
		return fmt.Errorf("certificate record save get certificate record %s: %s", t.fqdn, err)
	}
	if ct != nil {
		et = *ct
	}

	fqdn := []byte(t.fqdn)
	var bucket *bolt.Bucket

	// Expiration time index
	if t.ExpirationTime != et.ExpirationTime {
		if et.ExpirationTime != nil && !et.ExpirationTime.IsZero() {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexCertificateExpirationTimeFQDN,
				[]byte(et.ExpirationTime.Format(keyTimeLayout)+et.fqdn),
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		if t.ExpirationTime != nil && !t.ExpirationTime.IsZero() {
			if err := boltutils.BoltDeepPut(
				tx,
				bucketNameIndexCertificateExpirationTimeFQDN,
				[]byte(t.ExpirationTime.Format(keyTimeLayout)+t.fqdn),
				[]byte("1"),
			); err != nil {
				return fmt.Errorf("bolt deep put: %s", err)
			}
		}
	}

	// Save the certificate record data
	value, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNameCertificates)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameCertificates, err)
	}
	if err := bucket.Put(fqdn, value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameCertificates, t.fqdn, err)
	}

	return nil
}

func (t certificateRecord) delete(tx *bolt.Tx) (err error) {
	fqdn := []byte(t.fqdn)

	// Expiration time index
	if t.ExpirationTime != nil && !t.ExpirationTime.IsZero() {
		if err := boltutils.BoltDeepDelete(
			tx,
			bucketNameIndexCertificateExpirationTimeFQDN,
			[]byte(t.ExpirationTime.Format(keyTimeLayout)+t.fqdn),
		); err != nil {
			return fmt.Errorf("bolt deep delete: %s", err)
		}
	}

	// Certificate data
	bucket, err := tx.CreateBucketIfNotExists(bucketNameCertificates)
	if err != nil {
		return
	}
	return bucket.Delete(fqdn)
}

func getCertificates(tx *bolt.Tx, start []byte, limit int) (page *certificate.CertificatesPage, err error) {
	bucket := tx.Bucket(bucketNameCertificates)
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
	var r *certificateRecord
	for i = 0; k != nil && i < limit; i++ {
		r = &certificateRecord{}
		if err = json.Unmarshal(v, &r); err != nil {
			return
		}
		r.fqdn = string(k)
		e := r.export()
		page.Certificates = append(page.Certificates, *e)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}

func getCertificatesByExpiry(tx *bolt.Tx, since time.Time, start []byte, limit int) (page *certificate.InfosPage, err error) {
	bucket := tx.Bucket(bucketNameIndexCertificateExpirationTimeFQDN)
	if bucket != nil {
		return
	}
	c := bucket.Cursor()
	var k, _ []byte
	if len(start) == 0 {
		k, _ = c.First()
	} else {
		k, _ = c.Seek(start)
		var prev, p []byte
		for i := 0; i < limit; i++ {
			p, _ = c.Prev()
			if p == nil {
				break
			}
			prev = p
		}
		page.Previous = string(prev)
		k, _ = c.Seek(start)
	}
	var i int
	var r *certificateInfo
	for i = 0; k != nil && i < limit; {
		et, err := time.Parse(keyTimeLayout, string(k[:keyTimeLayoutLen]))
		if err != nil {
			return nil, err
		}
		if et.Before(since) {
			r, err = getCertificateInfo(tx, k[keyTimeLayoutLen:])
			if err != nil {
				return nil, err
			}
			e := r.export()
			page.Infos = append(page.Infos, *e)
			i++
		}
		k, _ = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}
