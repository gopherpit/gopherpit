// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltutils

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
)

// tempfile returns a temporary file path.
func tempfile() string {
	f, err := ioutil.TempFile("", "bolt-")
	if err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		panic(err)
	}
	return f.Name()
}

type DB struct {
	*bolt.DB
}

func (db DB) Destroy() {
	path := db.Path()
	if err := db.Close(); err != nil {
		panic(err)
	}
	if err := os.Remove(path); err != nil {
		panic(err)
	}
}

func NewDB(t *testing.T) DB {
	path := tempfile()
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		panic("db open error: " + err.Error())
	}
	t.Logf("bolt db file %s", db.Path())
	return DB{db}
}

func TestBoltDeep(t *testing.T) {
	db := NewDB(t)
	defer db.Destroy()

	bucket1Name := []byte("bucket1")
	bucket1NameFake := []byte("bucket1fake")
	bucket2Name := []byte("bucket2")
	bucket3Name := []byte("bucket3")
	keyName := []byte("key")
	putValue := []byte("value")

	t.Run("Put_Minimal", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepPut(tx, bucket1Name, keyName, putValue)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket1Name).Get(keyName)
			if !bytes.Equal(value, putValue) {
				t.Errorf("bucket %s key %s: expected %s, got %s", bucket1Name, keyName, putValue, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Put_Nested", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepPut(tx, bucket1Name, bucket2Name, keyName, putValue)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket1Name).Bucket(bucket2Name).Get(keyName)
			if !bytes.Equal(value, putValue) {
				t.Errorf("bucket %s.%s key %s: expected %s, got %s", bucket1Name, bucket2Name, keyName, putValue, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Put_Nested2", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepPut(tx, bucket3Name, bucket2Name, bucket1Name, keyName, putValue)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket3Name).Bucket(bucket2Name).Bucket(bucket1Name).Get(keyName)
			if !bytes.Equal(value, putValue) {
				t.Errorf("bucket %s.%s.%s key %s: expected %s, got %s", bucket3Name, bucket2Name, bucket1Name, keyName, putValue, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Put_Error", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepPut(tx, bucket1Name, keyName)
		}); err.Error() != "insufficient number of elements 2 < 3" {
			t.Errorf("invalid error %s", err)
		}
	})

	t.Run("Delete_Minimal", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket1Name, keyName)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket1Name).Get(keyName)
			if value != nil {
				t.Errorf("bucket %s key %s: expected nil, got %s", bucket1Name, keyName, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Delete_Minimal_No_Bucket", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket1NameFake, keyName)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}
	})

	t.Run("Delete_Minimal_No_Bucket_Nested", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket1Name, bucket1NameFake, keyName)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}
	})

	t.Run("Delete_Nested", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket1Name, bucket2Name, keyName)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket1Name).Bucket(bucket2Name).Get(keyName)
			if value != nil {
				t.Errorf("bucket %s.%s key %s: expected nil, got %s", bucket1Name, bucket2Name, keyName, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Delete_Nested2", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket3Name, bucket2Name, bucket1Name, keyName)
		}); err != nil {
			t.Fatalf("bolt db update transaction %s", err)
		}

		if err := db.DB.View(func(tx *bolt.Tx) error {
			value := tx.Bucket(bucket3Name).Bucket(bucket2Name).Bucket(bucket1Name).Get(keyName)
			if value != nil {
				t.Errorf("bucket %s.%s.%s key %s: expected nil, got %s", bucket3Name, bucket2Name, bucket1Name, keyName, value)
			}
			return nil
		}); err != nil {
			t.Fatalf("bolt db view transaction %s", err)
		}
	})

	t.Run("Delete_Error", func(t *testing.T) {
		if err := db.DB.Update(func(tx *bolt.Tx) error {
			return BoltDeepDelete(tx, bucket1Name)
		}); err.Error() != "insufficient number of elements 1 < 2" {
			t.Errorf("invalid error %s", err)
		}
	})
}
