// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/certificate"
)

var (
	bucketNameACMEChallenges = []byte("ACMEChallenges")
)

type acmeChallengeRecord struct {
	fqdn    string
	Token   string `json:"token,omitempty"`
	KeyAuth string `json:"key-auth,omitempty"`
}

func (t acmeChallengeRecord) export() *certificate.ACMEChallenge {
	return &certificate.ACMEChallenge{
		FQDN:    t.fqdn,
		Token:   t.Token,
		KeyAuth: t.KeyAuth,
	}
}

func (t *acmeChallengeRecord) update(o *certificate.ACMEChallengeOptions) error {
	if o.Token != nil {
		t.Token = *o.Token
	}
	if o.KeyAuth != nil {
		t.KeyAuth = *o.KeyAuth
	}
	return nil
}

func getACMEChallengeRecord(tx *bolt.Tx, fqdn []byte) (t *acmeChallengeRecord, err error) {
	bucket := tx.Bucket(bucketNameACMEChallenges)
	if bucket == nil {
		err = certificate.ACMEChallengeNotFound
		return
	}
	data := bucket.Get(fqdn)
	if data == nil {
		err = certificate.ACMEChallengeNotFound
		return
	}
	if err = json.Unmarshal(data, &t); err != nil {
		return
	}
	t.fqdn = string(fqdn)
	return
}

func (t *acmeChallengeRecord) save(tx *bolt.Tx) (err error) {
	t.fqdn = strings.ToLower(strings.TrimSpace(t.fqdn))
	// Required fields
	if t.fqdn == "" {
		return certificate.FQDNMissing
	}

	// Save the ACME Challenge record data
	value, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}
	bucket, err := tx.CreateBucketIfNotExists(bucketNameACMEChallenges)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameACMEChallenges, err)
	}
	if err := bucket.Put([]byte(t.fqdn), value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameACMEChallenges, t.fqdn, err)
	}

	return nil
}

func (t acmeChallengeRecord) delete(tx *bolt.Tx) (err error) {
	// ACME Challenge data
	bucket, err := tx.CreateBucketIfNotExists(bucketNameACMEChallenges)
	if err != nil {
		return
	}
	return bucket.Delete([]byte(t.fqdn))
}

func getACMEChallenges(tx *bolt.Tx, start []byte, limit int) (page *certificate.ACMEChallengesPage, err error) {
	bucket := tx.Bucket(bucketNameACMEChallenges)
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
	var r *acmeChallengeRecord
	for i = 0; k != nil && i < limit; i++ {
		r = &acmeChallengeRecord{}
		if err = json.Unmarshal(v, &r); err != nil {
			return
		}
		r.fqdn = string(k)
		e := r.export()
		page.ACMEChallenges = append(page.ACMEChallenges, *e)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}
