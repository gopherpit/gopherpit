package fileServer

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"strings"
)

var hexChars = []rune("0123456789abcdef")

type Hasher interface {
	Hash(io.Reader) (string, error)
	IsHash(string) bool
}

type MD5Hasher struct {
	HashLength int
}

func (s MD5Hasher) Hash(reader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	h := hash.Sum(nil)
	if len(h) < s.HashLength {
		return "", nil
	}
	return strings.TrimRight(hex.EncodeToString(h)[:s.HashLength], "="), nil
}

func (s MD5Hasher) IsHash(h string) bool {
	if len(h) != s.HashLength {
		return false
	}
	var found bool
	for _, c := range h {
		found = false
		for _, m := range hexChars {
			if c == m {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
