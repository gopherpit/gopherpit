// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltPackages

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/pkg/boltutils"

	"gopherpit.com/gopherpit/services/packages"
)

var (
	bucketNameDomains                      = []byte("Domains")
	bucketNameIndexFQDNDomainID            = []byte("Index_FQDN_DomainID")
	bucketNameIndexOwnerUserIDFQDNDomainID = []byte("Index_OwnerUserID_FQDN_DomainID")
	bucketNameIndexDisabledDomainIDs       = []byte("Index_Disabled_DomainID_1")

	bucketNameIndexDomainIDUserIDs    = []byte("Index_DomainID_UserID_1")
	bucketNameIndexUserIDFQDNDomainID = []byte("Index_UserID_FQDN_DomainID")

	flagBytes      = []byte("1")
	base32Encoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz")
)

type domainRecord struct {
	id                string
	FQDN              string `json:"fqdn"`
	OwnerUserID       string `json:"owner-user-id,omitempty"`
	CertificateIgnore bool   `json:"certificate-ignore,omitempty"`
	Disabled          bool   `json:"disabled,omitempty"`

	// Internal functionality fields (changes not logged)
	CertificateIgnoreMissing bool `json:"certificate-ignore-missing,omitempty"`
}

func (d domainRecord) export() *packages.Domain {
	return &packages.Domain{
		ID:                d.id,
		FQDN:              d.FQDN,
		OwnerUserID:       d.OwnerUserID,
		CertificateIgnore: d.CertificateIgnore,
		Disabled:          d.Disabled,

		CertificateIgnoreMissing: d.CertificateIgnoreMissing,
	}
}

func (d *domainRecord) update(o *packages.DomainOptions) (changes packages.Changes) {
	if o.FQDN != nil {
		if *o.FQDN != d.FQDN {
			changes = append(changes, packages.Change{
				Field: "fqdn",
				From:  stringToStringPtr(d.FQDN),
				To:    stringToStringPtr(*o.FQDN),
			})
		}
		d.FQDN = *o.FQDN
	}
	if o.OwnerUserID != nil {
		if *o.OwnerUserID != d.OwnerUserID {
			changes = append(changes, packages.Change{
				Field: "owner-user-id",
				From:  stringToStringPtr(d.OwnerUserID),
				To:    stringToStringPtr(*o.OwnerUserID),
			})
		}
		d.OwnerUserID = *o.OwnerUserID
	}
	if o.CertificateIgnore != nil {
		if *o.CertificateIgnore != d.CertificateIgnore {
			changes = append(changes, packages.Change{
				Field: "certificate-ignore",
				From:  boolPtrToStringPtr(&d.CertificateIgnore),
				To:    boolPtrToStringPtr(o.CertificateIgnore),
			})
		}
		d.CertificateIgnore = *o.CertificateIgnore
	}
	if o.Disabled != nil {
		if *o.Disabled != d.Disabled {
			changes = append(changes, packages.Change{
				Field: "disabled",
				From:  boolPtrToStringPtr(&d.Disabled),
				To:    boolPtrToStringPtr(o.Disabled),
			})
		}
		d.Disabled = *o.Disabled
	}

	// Internal functionality fields (changes not logged)
	if o.CertificateIgnoreMissing != nil {
		d.CertificateIgnoreMissing = *o.CertificateIgnoreMissing
	}
	return
}

func getDomainRecord(tx *bolt.Tx, id []byte) (d *domainRecord, err error) {
	d, err = getDomainRecordByID(tx, id)
	if err == packages.DomainNotFound {
		d, err = getDomainRecordByFQDN(tx, id)
	}
	return
}

func getDomainRecordByID(tx *bolt.Tx, id []byte) (d *domainRecord, err error) {
	bucket := tx.Bucket(bucketNameDomains)
	if bucket == nil {
		err = packages.DomainNotFound
		return
	}
	data := bucket.Get(id)
	if data == nil {
		err = packages.DomainNotFound
		return
	}
	if err = json.Unmarshal(data, &d); err != nil {
		return
	}
	d.id = string(id)
	return
}

func getDomainRecordByFQDN(tx *bolt.Tx, fqdn []byte) (d *domainRecord, err error) {
	bucket := tx.Bucket(bucketNameIndexFQDNDomainID)
	if bucket == nil {
		err = packages.DomainNotFound
		return
	}

	id := bucket.Get(fqdn)
	if id == nil {
		err = packages.DomainNotFound
		return
	}

	d, err = getDomainRecordByID(tx, id)
	return
}

func isDomainDisabled(tx *bolt.Tx, id []byte) (disabled bool, err error) {
	bucket := tx.Bucket(bucketNameIndexDisabledDomainIDs)
	if bucket != nil {
		disabled = bucket.Get(id) != nil
	}
	return
}

func getDomainIDByFQDN(tx *bolt.Tx, fqdn []byte) (id []byte, err error) {
	bucket := tx.Bucket(bucketNameIndexFQDNDomainID)
	if bucket == nil {
		err = packages.DomainNotFound
		return
	}
	id = bucket.Get(fqdn)
	if id == nil {
		err = packages.DomainNotFound
		return
	}
	return
}

func (d *domainRecord) save(tx *bolt.Tx) (err error) {
	// Fields validation
	d.FQDN = strings.TrimSpace(strings.ToLower(d.FQDN))
	if d.FQDN == "" {
		return packages.DomainFQDNRequired
	}

	if d.OwnerUserID == "" {
		return packages.DomainOwnerUserIDRequired
	}

	fqdn := []byte(d.FQDN)

	// Existing Domain record
	ed := &domainRecord{}
	if d.id == "" {
		// Generate new id
		id, err := newDomainID(tx)
		if err != nil {
			return fmt.Errorf("generate unique ID: %s", err)
		}
		d.id = id
	} else {
		// Check if domain with d.ID exists
		cd, err := getDomainRecordByID(tx, []byte(d.id))
		if err != nil {
			return fmt.Errorf("get domain record %s: %s", d.id, err)
		}
		if cd != nil {
			ed = cd
		}
	}

	id := []byte(d.id)

	// FQDN must be unique
	if d.FQDN != "" {
		ci, err := getDomainIDByFQDN(tx, fqdn)
		switch err {
		case packages.DomainNotFound:
		case nil:
			if !bytes.Equal(ci, id) {
				return packages.DomainAlreadyExists
			}
		default:
			return fmt.Errorf("get domain id by fqdn: %s", err)
		}
	}

	var bucket *bolt.Bucket

	// FQDN index
	if d.FQDN != ed.FQDN {
		bucket, err = tx.CreateBucketIfNotExists(bucketNameIndexFQDNDomainID)
		if err != nil {
			return
		}
		if ed.FQDN != "" {
			if err := bucket.Delete([]byte(ed.FQDN)); err != nil {
				return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexFQDNDomainID, ed.FQDN, err)
			}

			// Combined UserID FQDN index
			b0 := tx.Bucket(bucketNameIndexDomainIDUserIDs)
			if b0 != nil {
				b0 = b0.Bucket(id)
			}
			if b0 != nil {
				b1 := tx.Bucket(bucketNameIndexUserIDFQDNDomainID)
				var b2 *bolt.Bucket
				if err = b0.ForEach(func(userID, _ []byte) error {
					b2 = b1.Bucket(userID)
					if b2 != nil {
						if err := b0.Delete([]byte(ed.FQDN)); err != nil {
							return fmt.Errorf("bucket(%s).bucket(%s).Delete(%s): %s", bucketNameIndexUserIDFQDNDomainID, userID, d.FQDN, err)
						}
					}
					return nil
				}); err != nil {
					return
				}
			}
		}
		if d.FQDN != "" {
			if err := bucket.Put(fqdn, id); err != nil {
				return fmt.Errorf("bucket(%s).Put(%s): %s", bucketNameIndexFQDNDomainID, d.FQDN, err)
			}

			// Combined UserID FQDN index
			b0 := tx.Bucket(bucketNameIndexDomainIDUserIDs)
			if b0 != nil {
				b0 = b0.Bucket(id)
			}
			if b0 != nil {
				b1 := tx.Bucket(bucketNameIndexUserIDFQDNDomainID)
				var b2 *bolt.Bucket
				if err = b0.ForEach(func(userID, _ []byte) error {
					b2 = b1.Bucket(userID)
					if b2 != nil {
						if err := b0.Put(fqdn, id); err != nil {
							return fmt.Errorf("bucket(%s).bucket(%s).Delete(%s): %s", bucketNameIndexUserIDFQDNDomainID, userID, d.FQDN, err)
						}
					}
					return nil
				}); err != nil {
					return
				}
			}
		}
	}

	if d.OwnerUserID != ed.OwnerUserID || d.FQDN != ed.FQDN {
		// Combined OwnerUserID FQDN index
		if ed.FQDN != "" && ed.OwnerUserID != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexOwnerUserIDFQDNDomainID,
				[]byte(ed.OwnerUserID),
				[]byte(ed.FQDN),
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		if d.FQDN != "" && d.OwnerUserID != "" {
			if err := boltutils.BoltDeepPut(
				tx,
				bucketNameIndexOwnerUserIDFQDNDomainID,
				[]byte(d.OwnerUserID),
				[]byte(d.FQDN),
				id,
			); err != nil {
				return fmt.Errorf("bolt deep put: %s", err)
			}
		}

		// Combined UserID FQDN index
		if ed.FQDN != "" && ed.OwnerUserID != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexUserIDFQDNDomainID,
				[]byte(ed.OwnerUserID),
				[]byte(ed.FQDN),
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		if d.FQDN != "" && d.OwnerUserID != "" {
			if err := boltutils.BoltDeepPut(
				tx,
				bucketNameIndexUserIDFQDNDomainID,
				[]byte(d.OwnerUserID),
				[]byte(d.FQDN),
				id,
			); err != nil {
				return fmt.Errorf("bolt deep put: %s", err)
			}
		}
	}

	if ed.OwnerUserID != "" && d.OwnerUserID != ed.OwnerUserID {
		if err = d.addUser(tx, []byte(ed.OwnerUserID), false); err != nil {
			return fmt.Errorf("add previous owner as user: %s", err)
		}
	}

	// Disabled index
	if d.Disabled == false && ed.Disabled == true {
		bucket = tx.Bucket(bucketNameIndexDisabledDomainIDs)
		if bucket != nil {
			if err := bucket.Delete(id); err != nil {
				return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexDisabledDomainIDs, d.id, err)
			}
		}
	}
	if d.Disabled == true && ed.Disabled == false {
		bucket = tx.Bucket(bucketNameIndexDisabledDomainIDs)
		if bucket != nil {
			if err := bucket.Put(id, flagBytes); err != nil {
				return fmt.Errorf("bucket(%s).Put(%s, %s) %s", bucketNameIndexDisabledDomainIDs, d.id, flagBytes, err)
			}
		}
	}

	// Save the domain record data
	value, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNameDomains)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNameDomains, err)
	}
	if err := bucket.Put(id, value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNameDomains, d.id, err)
	}
	return
}

func (d domainRecord) delete(tx *bolt.Tx) (err error) {
	id := []byte(d.id)
	var bucket *bolt.Bucket

	// FQDN index
	if d.FQDN != "" {
		fqdn := []byte(d.FQDN)
		bucket = tx.Bucket(bucketNameIndexFQDNDomainID)
		if bucket != nil {
			if err := bucket.Delete(fqdn); err != nil {
				return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexFQDNDomainID, d.FQDN, err)
			}
		}
		// Combined OwnerUserID FQDN index
		if d.OwnerUserID != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexOwnerUserIDFQDNDomainID,
				[]byte(d.OwnerUserID),
				fqdn,
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		// Combined UserID FQDN index
		if d.OwnerUserID != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexUserIDFQDNDomainID,
				[]byte(d.OwnerUserID),
				fqdn,
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}

		// Combined UserID FQDN index
		bucket = tx.Bucket(bucketNameIndexDomainIDUserIDs)
		if bucket != nil {
			bucket = bucket.Bucket(id)
		}
		if bucket != nil {
			b1 := tx.Bucket(bucketNameIndexUserIDFQDNDomainID)
			var b2 *bolt.Bucket
			if err = bucket.ForEach(func(userID, _ []byte) error {
				b2 = b1.Bucket(userID)
				if b2 != nil {
					if err := b2.Delete(fqdn); err != nil {
						return fmt.Errorf("bucket(%s).bucket(%s).Delete(%s): %s", bucketNameIndexUserIDFQDNDomainID, userID, d.FQDN, err)
					}
				}
				return nil
			}); err != nil {
				return
			}
		}
	}

	// DomainID UserIDs index
	bucket = tx.Bucket(bucketNameIndexDomainIDUserIDs)
	if bucket != nil {
		if err := bucket.DeleteBucket(id); err != nil && err != bolt.ErrBucketNotFound {
			return fmt.Errorf("bucket(%s).DeleteBucket(%s): %s", bucketNameIndexDomainIDUserIDs, d.id, err)
		}
	}

	// Disabled index
	bucket = tx.Bucket(bucketNameIndexDisabledDomainIDs)
	if bucket != nil {
		if err := bucket.Delete(id); err != nil {
			return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexDisabledDomainIDs, d.id, err)
		}
	}

	// Package records
	if bucket = tx.Bucket(bucketNameIndexDomainIDPathPackageID); bucket != nil {
		if bucket = bucket.Bucket(id); bucket != nil {
			if err = bucket.ForEach(func(k, id []byte) error {
				p, err := getPackageRecord(tx, id)
				if err != nil {
					return err
				}
				return p.delete(tx)
			}); err != nil {
				return
			}
		}
	}

	// Domain data
	bucket, err = tx.CreateBucketIfNotExists(bucketNameDomains)
	if err != nil {
		return
	}
	return bucket.Delete(id)
}

func (d domainRecord) addUser(tx *bolt.Tx, userID []byte, checkExists bool) (err error) {
	// DomainID UserIDs index
	bucket, err := tx.CreateBucketIfNotExists(bucketNameIndexDomainIDUserIDs)
	if err != nil {
		return
	}
	id := []byte(d.id)
	bucket, err = bucket.CreateBucketIfNotExists(id)
	if err != nil {
		return
	}
	if checkExists && bucket.Get(userID) != nil {
		err = packages.UserExists
		return
	}
	if err = bucket.Put(userID, flagBytes); err != nil {
		return
	}

	// Combined UserID FQDN index
	bucket, err = tx.CreateBucketIfNotExists(bucketNameIndexUserIDFQDNDomainID)
	if err != nil {
		return
	}
	bucket, err = bucket.CreateBucketIfNotExists(userID)
	if err != nil {
		return
	}
	err = bucket.Put([]byte(d.FQDN), id)
	return
}

func (d domainRecord) removeUser(tx *bolt.Tx, userID []byte) (err error) {
	bucket := tx.Bucket(bucketNameIndexDomainIDUserIDs)
	if bucket == nil {
		err = packages.UserDoesNotExist
		return
	}
	bucket = bucket.Bucket([]byte(d.id))
	if bucket == nil {
		err = packages.UserDoesNotExist
		return
	}
	if bucket.Get(userID) == nil {
		err = packages.UserDoesNotExist
		return
	}
	if err = bucket.Delete(userID); err != nil {
		return
	}
	if err := boltutils.BoltDeepDelete(
		tx,
		bucketNameIndexUserIDFQDNDomainID,
		userID,
		[]byte(d.FQDN),
	); err != nil {
		return fmt.Errorf("bolt deep delete: %s", err)
	}
	return nil
}

func (d domainRecord) isOwner(id string) bool {
	return d.OwnerUserID == id
}

func (d domainRecord) isUser(tx *bolt.Tx, id string) bool {
	if d.isOwner(id) {
		return true
	}
	bucket := tx.Bucket(bucketNameIndexDomainIDUserIDs)
	if bucket != nil {
		bucket = bucket.Bucket([]byte(d.id))
		if bucket != nil {
			if bucket.Get([]byte(id)) != nil {
				return true
			}
		}
	}
	return false
}

func newDomainID(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 19)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNameDomains)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		_, err = rand.Read(br)
		if err != nil {
			return
		}
		b := append(bt, append(bp, br...)...)
		id = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		if v := bucket.Get([]byte(id)); v == nil {
			return
		}
	}
	return "", errors.New("unable to generate unique domain id")
}
