// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltPackages

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/packages"
)

var (
	bucketNameChangelogDomainID = []byte("Changelog_DomainID")

	keyTimeLayout    = "20060102150405.000000000"
	keyTimeLayoutLen = len(keyTimeLayout)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type changelogRecord struct {
	id        string
	time      time.Time
	domainID  string
	FQDN      string           `json:"fqdn,omitempty"`
	PackageID string           `json:"package-id,omitempty"`
	Path      string           `json:"path,omitempty"`
	UserID    string           `json:"user-id,omitempty"`
	Action    packages.Action  `json:"action,omitempty"`
	Changes   packages.Changes `json:"changes,omitempty"`
}

func (r changelogRecord) export() (cr *packages.ChangelogRecord, err error) {
	time, err := r.getTime()
	if err != nil {
		return
	}
	return &packages.ChangelogRecord{
		ID:        r.id,
		Time:      time,
		DomainID:  r.domainID,
		FQDN:      r.FQDN,
		PackageID: r.PackageID,
		Path:      r.Path,
		UserID:    r.UserID,
		Action:    r.Action,
		Changes:   r.Changes,
	}, nil
}

func (r changelogRecord) newID(tx *bolt.Tx) (id string, err error) {
	if r.time.IsZero() {
		r.time = time.Now()
	}
	time := r.time.UTC()
	var v []byte
	timestamp := time.Format(keyTimeLayout)
	bucket := tx.Bucket(bucketNameChangelogDomainID)
	if bucket != nil {
		bucket = bucket.Bucket([]byte(r.domainID))
	}
	for i := 0; i < 20; i++ {
		id = fmt.Sprintf("%s%03d", timestamp, rand.Intn(1000))
		if bucket != nil {
			v = bucket.Get([]byte(id))
			if v == nil {
				break
			}
		}
	}
	if v != nil {
		err = errors.New("could not generate unique id")
	}
	return
}

func (r changelogRecord) getTime() (t time.Time, err error) {
	if !r.time.IsZero() {
		t = r.time
		return
	}
	return timeFromID(r.id)
}

func timeFromID(id string) (t time.Time, err error) {
	if len(id) >= keyTimeLayoutLen {
		return time.Parse(keyTimeLayout, id[:keyTimeLayoutLen])
	}
	err = errors.New("id is too short")
	return
}

func getChangelogRecord(tx *bolt.Tx, domainID, id []byte) (r *changelogRecord, err error) {
	r = &changelogRecord{
		id: string(id),
	}
	bucket := tx.Bucket(bucketNameChangelogDomainID)
	if bucket == nil {
		err = packages.ErrChangelogRecordNotFound
		return
	}
	bucket = bucket.Bucket(domainID)
	if bucket == nil {
		err = packages.ErrDomainNotFound
		return
	}
	data := bucket.Get(id)
	if data == nil {
		err = packages.ErrChangelogRecordNotFound
		return
	}
	if err = json.Unmarshal(data, &r); err != nil {
		return
	}
	r.id = string(id)
	r.domainID = string(domainID)
	return
}

func (r *changelogRecord) save(tx *bolt.Tx) (err error) {
	if r.id == "" {
		r.id, err = r.newID(tx)
		if err != nil {
			return fmt.Errorf("generate unique ID: %s", err)
		}
	}
	if r.domainID == "" {
		return errors.New("domain id missing")
	}

	value, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}
	bucket, err := tx.CreateBucketIfNotExists(bucketNameChangelogDomainID)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s): %s", bucketNameChangelogDomainID, err)
	}
	bucket, err = bucket.CreateBucketIfNotExists([]byte(r.domainID))
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s): %s", r.domainID, err)
	}
	if err := bucket.Put([]byte(r.id), value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s): %s", bucketNameChangelogDomainID, r.id, err)
	}
	return nil
}

func (r *changelogRecord) delete(tx *bolt.Tx) (err error) {
	bucket := tx.Bucket(bucketNameChangelogDomainID)
	if bucket == nil {
		return
	}
	bucket = bucket.Bucket([]byte(r.domainID))
	if bucket == nil {
		return
	}
	if err := bucket.Delete([]byte(r.id)); err != nil {
		return err
	}
	r = &changelogRecord{}
	return
}

type chagelogRecordData struct {
	domainID  string
	fqdn      string
	packageID string
	path      string
	userID    string
	action    packages.Action
	changes   packages.Changes
}

func (s Service) newChangelogRecord(data chagelogRecordData) error {
	now := time.Now().UTC()
	c, err := s.Changelog.NewConnection(now)
	if err != nil {
		return err
	}
	defer c.Close()

	if err = c.DB.Update(func(tx *bolt.Tx) (err error) {
		return (&changelogRecord{
			time:      now,
			domainID:  data.domainID,
			FQDN:      data.fqdn,
			PackageID: data.packageID,
			Path:      data.path,
			UserID:    data.userID,
			Action:    data.action,
			Changes:   data.changes,
		}).save(tx)
	}); err != nil {
		return err
	}
	return nil
}

func boolPtrToStringPtr(b *bool) *string {
	t := "true"
	f := "false"
	if b == nil {
		return &f
	}
	if *b == false {
		return &f
	}
	return &t
}

func stringToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
