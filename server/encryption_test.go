// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	keyLen := 16
	key := make([]byte, 16)
	n, err := rand.Read(key)
	if err != nil {
		t.Errorf("rand.Read: %s", err)
	}
	if n != keyLen {
		t.Errorf("rand.Read: only %d bytes read out of %d", n, keyLen)
	}
	data := []byte("testing")
	enc, err := encrypt(key, data)
	if err != nil {
		t.Errorf("encrypt: %s", err)
	}
	dec, err := decrypt(key, enc)
	if err != nil {
		t.Errorf("encrypt: %s", err)
	}
	if !bytes.Equal(data, dec) {
		t.Errorf("Original and decrypted data are not equal")
	}
}
