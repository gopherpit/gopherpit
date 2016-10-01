// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
)

func encrypt(key, text []byte) ([]byte, error) {
	bytetext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %s", err)
	}

	ciphertext := make([]byte, aes.BlockSize+len(bytetext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("io read full: %s", err)
	}

	cipher.NewCFBEncrypter(block, iv).XORKeyStream(ciphertext[aes.BlockSize:], bytetext)
	buf := make([]byte, base32.StdEncoding.EncodedLen(len(ciphertext)))
	base32.StdEncoding.Encode(buf, ciphertext)
	return bytes.ToLower(bytes.TrimRight(buf, "=")), nil
}

func decrypt(key, text []byte) ([]byte, error) {
	text = bytes.ToUpper(text)
	if len(text)%8 != 0 {
		text = append(text, bytes.Repeat([]byte("="), 8-len(text)%8)...)
	}
	ciphertext := make([]byte, base32.StdEncoding.DecodedLen(len(text)))
	n, err := base32.StdEncoding.Decode(ciphertext, text)
	if err != nil {
		return nil, fmt.Errorf("base32 decode: %s", err)
	}
	ciphertext = ciphertext[:n]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %s", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	cipher.NewCFBDecrypter(block, iv).XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
