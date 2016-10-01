// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

// ErrInvalidDuration is error returned if no duration can be
// encoded or decoded.
var ErrInvalidDuration = errors.New("invalid duration")

// Duration is a helper type that wraps time.Duration
// to JSON encode and decode durations in humanly readable form.
type Duration time.Duration

// MarshalJSON implements json.Marshaler interface.
// It marshals a string representation of time.Duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration().String())
}

// UnmarshalJSON implements json.Unamrshaler interface.
// It parses time.Duration string.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if len(data) <= 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return ErrInvalidDuration
	}
	if err := d.UnmarshalText(bytes.Trim(data, "\"")); err != nil {
		return ErrInvalidDuration
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler interface.
// It marshals a string representation of time.Duration.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration().String()), nil
}

// UnmarshalText implements encoding.TextUnamrshaler interface.
// It parses time.Duration string.
func (d *Duration) UnmarshalText(data []byte) error {
	x, err := time.ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}

// Duration returns time.Duration of a Duration value.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}
