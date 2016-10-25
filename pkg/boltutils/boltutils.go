// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltutils

import (
	"fmt"

	"github.com/boltdb/bolt"
)

// BoltDeepPut saves the last element of elements arguments under the
// key named by the second to last element, and all previous elements
// will be created as buckets if any of them do not exist.
func BoltDeepPut(tx *bolt.Tx, elements ...[]byte) (err error) {
	length := len(elements)
	if length < 3 {
		return fmt.Errorf("insufficient number of elements %d < 3", length)
	}
	path := elements[0]
	bucket, err := tx.CreateBucketIfNotExists(elements[0])
	if err != nil {
		return fmt.Errorf("bucket create %s: %s", elements[0], err)
	}
	for i := 1; i < length-2; i++ {
		path = append(path, []byte(", ")...)
		path = append(path, elements[i]...)
		bucket, err = bucket.CreateBucketIfNotExists(elements[i])
		if err != nil {
			return fmt.Errorf("bucket create %s: %s", path, err)
		}
	}
	if err = bucket.Put(elements[length-2], elements[length-1]); err != nil {
		return fmt.Errorf("bucket %s put %s: %s", path, elements[length-2], err)
	}
	return
}

// BoltDeepDelete deletes the key named as the last element of the elements
// arguments in nested buckets named as previous elements.
func BoltDeepDelete(tx *bolt.Tx, elements ...[]byte) (err error) {
	length := len(elements)
	if length < 2 {
		return fmt.Errorf("insufficient number of elements %d < 2", length)
	}
	path := elements[0]
	bucket := tx.Bucket(elements[0])
	if bucket == nil {
		return
	}
	for i := 1; i < length-1; i++ {
		path = append(path, []byte(", ")...)
		path = append(path, elements[i]...)
		bucket = bucket.Bucket(elements[i])
		if bucket == nil {
			return
		}
	}
	if err = bucket.Delete(elements[length-1]); err != nil {
		return fmt.Errorf("bucket %s delete %s: %s", path, elements[length-1], err)
	}
	return
}
