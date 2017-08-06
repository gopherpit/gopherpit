// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
)

// ErrInvalidURL is error returned if no URL can be
// encoded or decoded.
var ErrInvalidURL = errors.New("invalid url")

// URL is a helper type that wraps url.URL
// to JSON encode and decode URLs.
type URL url.URL

// MarshalJSON implements json.Marshaler interface.
// It marshals a string representation of url.URL.
func (u URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.URL().String())
}

// UnmarshalJSON implements json.Unmarshaler interface.
// It parses url.URL string.
func (u *URL) UnmarshalJSON(data []byte) error {
	if len(data) <= 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return ErrInvalidURL
	}
	if err := u.UnmarshalText(bytes.Trim(data, "\"")); err != nil {
		return ErrInvalidURL
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler interface.
// It marshals a string representation of url.URL.
func (u URL) MarshalText() ([]byte, error) {
	return []byte(u.URL().String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler interface.
// It parses url.URL string.
func (u *URL) UnmarshalText(data []byte) error {
	x, err := url.Parse(string(data))
	if err != nil {
		return err
	}
	if x == nil {
		return ErrInvalidURL
	}
	*u = URL(*x)
	return nil
}

// URL returns url.URL of a URL value.
func (u URL) URL() *url.URL {
	x := url.URL(u)
	return &x
}
