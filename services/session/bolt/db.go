// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltSession

import (
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/session"
)

var (
	bucketNameSessions = []byte("Sessions")

	defaultLifetime = 60 * 24 * time.Hour
)

type sessionRecord struct {
	id      string
	Values  map[string]interface{} `json:"values,omitempty"`
	MaxAge  int                    `json:"max-age,omitempty"`
	Expires time.Time              `json:"expires,omitempty"`
}

func (s sessionRecord) export() *session.Session {
	return &session.Session{
		ID:     s.id,
		Values: s.Values,
		MaxAge: s.MaxAge,
	}
}

func (s *sessionRecord) update(data session.Options) {
	if data.Values != nil {
		s.Values = *data.Values
	}
	if data.MaxAge != nil {
		s.MaxAge = *data.MaxAge
	}
}

func getSessionRecord(tx *bolt.Tx, id []byte) (s *sessionRecord, err error) {
	s = &sessionRecord{
		id: string(id),
	}
	bucket := tx.Bucket(bucketNameSessions)
	if bucket == nil {
		err = session.SessionNotFound
		return
	}
	data := bucket.Get(id)
	if data == nil {
		err = session.SessionNotFound
		return
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return
	}
	s.id = string(id)
	return
}

func (s *sessionRecord) save(tx *bolt.Tx, lifetime time.Duration) (err error) {
	if s.id == "" {
		id, err := newSessionID(tx)
		if err != nil {
			return fmt.Errorf("session record save generate unique ID: %s", err)
		}
		s.id = id
	}

	switch {
	case s.MaxAge > 0:
		s.Expires = time.Now().Add(time.Second * time.Duration(s.MaxAge))
	case lifetime > 0:
		s.Expires = time.Now().Add(lifetime)
	default:
		s.Expires = time.Now().Add(defaultLifetime)
	}

	value, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("session record save json.Marshal %s", err)
	}
	bucket, err := tx.CreateBucketIfNotExists(bucketNameSessions)
	if err != nil {
		return fmt.Errorf("session record save CreateBucketIfNotExists(%s) %s", bucketNameSessions, err)
	}
	if err := bucket.Put([]byte(s.id), value); err != nil {
		return fmt.Errorf("session record save bucket(%s).Put(%s) %s", bucketNameSessions, s.id, err)
	}

	return nil
}

func delete(id []byte, tx *bolt.Tx) (err error) {
	bucket := tx.Bucket(bucketNameSessions)
	if bucket == nil {
		return
	}
	if v := bucket.Get(id); v == nil {
		return session.SessionNotFound
	}
	if err := bucket.Delete(id); err != nil {
		return err
	}
	return
}

func sessionCount(tx *bolt.Tx) (count int) {
	bucket := tx.Bucket(bucketNameSessions)
	if bucket == nil {
		return 0
	}
	return bucket.Stats().KeyN
}

var base32Encoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz")

func newSessionID(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 19)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNameSessions)
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
	return "", errors.New("unable to generate unique session id")
}
