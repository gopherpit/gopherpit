// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltKey

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"gopherpit.com/gopherpit/services/key"
)

var (
	bucketNameKeys           = []byte("Keys")
	bucketNameIndexRefSecret = []byte("Index_Ref_Secret")
)

type keyRecord struct {
	secret             string
	Ref                string      `json:"ref,omitempty"`
	AuthorizedNetworks []net.IPNet `json:"authorized-networks,omitempty"`
}

func (r keyRecord) export() *key.Key {
	return &key.Key{
		Secret:             r.secret,
		Ref:                r.Ref,
		AuthorizedNetworks: r.AuthorizedNetworks,
	}
}

func (r *keyRecord) update(o *key.Options) {
	if o == nil {
		return
	}
	if o.AuthorizedNetworks != nil {
		r.AuthorizedNetworks = *o.AuthorizedNetworks
	}
}

func getKeyRecordBySecret(tx *bolt.Tx, secret []byte) (r *keyRecord, err error) {
	bucket := tx.Bucket(bucketNameKeys)
	if bucket == nil {
		err = key.ErrKeyNotFound
		return
	}
	data := bucket.Get(secret)
	if data == nil {
		err = key.ErrKeyNotFound
		return
	}
	if err = json.Unmarshal(data, &r); err != nil {
		return
	}
	r.secret = string(secret)
	return
}

func getKeyRecordByRef(tx *bolt.Tx, ref []byte) (r *keyRecord, err error) {
	bucket := tx.Bucket(bucketNameIndexRefSecret)
	if bucket == nil {
		err = key.ErrKeyNotFound
		return
	}
	secret := bucket.Get(ref)
	if secret == nil {
		err = key.ErrKeyNotFound
		return
	}
	return getKeyRecordBySecret(tx, secret)
}

func (r *keyRecord) save(tx *bolt.Tx, secret string) (err error) {
	// Required fields
	if r.Ref == "" {
		return key.ErrKeyRefRequired
	}

	// existing key record
	re := &keyRecord{}
	if r.secret == "" {
		// Generate new secret if it is not provided and it is a new record
		if secret == "" {
			secret, err = newKeySecret(tx)
			if err != nil {
				return
			}
		}
	} else {
		// Check if key with r.secret exists
		rc, err := getKeyRecordBySecret(tx, []byte(r.secret))
		if err != nil {
			return fmt.Errorf("get key record by secret: %s", err)
		}
		if rc != nil {
			re = rc
		}
	}

	// Set secret if it is provided or it is a new record
	if secret != "" {
		r.secret = secret
	}

	// Save ref index
	bucket, err := tx.CreateBucketIfNotExists(bucketNameIndexRefSecret)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s): %s", bucketNameIndexRefSecret, err)
	}
	if err := bucket.Put([]byte(r.Ref), []byte(r.secret)); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameIndexRefSecret, r.Ref, err)
	}

	// Prepare the data
	value, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	bucket, err = tx.CreateBucketIfNotExists(bucketNameKeys)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s): %s", bucketNameKeys, err)
	}

	// Clean existing key record if secret has been changed
	if re.secret != r.secret {
		if err = bucket.Delete([]byte(re.secret)); err != nil {
			return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameKeys, "[secret]", err)
		}
	}

	// Save user data
	if err := bucket.Put([]byte(r.secret), value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s): %s", bucketNameKeys, "[secret]", err)
	}

	return nil
}

func delete(tx *bolt.Tx, ref []byte) (err error) {
	var r *keyRecord
	r, err = getKeyRecordByRef(tx, ref)
	if err != nil {
		return
	}
	if bucket := tx.Bucket(bucketNameKeys); bucket != nil {
		if err := bucket.Delete([]byte(r.secret)); err != nil {
			return err
		}
	}
	if bucket := tx.Bucket(bucketNameIndexRefSecret); bucket != nil {
		if err := bucket.Delete([]byte(r.Ref)); err != nil {
			return err
		}
	}
	return
}

func regenerateSecret(tx *bolt.Tx, ref []byte) (secret string, err error) {
	var r *keyRecord
	r, err = getKeyRecordByRef(tx, ref)
	if err != nil {
		return
	}
	secret, err = newKeySecret(tx)
	if err != nil {
		return
	}
	if err = r.save(tx, secret); err != nil {
		return
	}
	secret = r.secret
	return
}

func getKeys(tx *bolt.Tx, start []byte, limit int) (page key.KeysPage, err error) {
	indexBucket := tx.Bucket(bucketNameIndexRefSecret)
	if indexBucket == nil {
		return
	}
	c := indexBucket.Cursor()
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
	bucket := tx.Bucket(bucketNameKeys)
	if bucket != nil {
		return
	}
	var i int
	var r *keyRecord
	for i = 0; k != nil && i < limit; i++ {
		data := bucket.Get(v)
		if data == nil {
			i--
			continue
		}
		if err = json.Unmarshal(data, &r); err != nil {
			return
		}
		r.secret = string(v)
		x := r.export()
		page.Keys = append(page.Keys, *x)
		k, v = c.Next()
	}
	page.Next = string(k)
	page.Count = int(i)
	return
}

var base32Encoding = base32.NewEncoding("ABCDEFGHJKMNPQRSTVWXYZ0123456789")

func newKeySecret(tx *bolt.Tx) (secret string, err error) {
	br1 := make([]byte, 4)
	br2 := make([]byte, 14)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNameKeys)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		if _, err = rand.Read(br1); err != nil {
			return
		}
		if _, err = rand.Read(br2); err != nil {
			return
		}
		b := append(br1, append(bt, br2...)...)
		secret = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		if v := bucket.Get([]byte(secret)); v == nil {
			return
		}
	}
	return "", errors.New("unable to generate unique key secret")
}
