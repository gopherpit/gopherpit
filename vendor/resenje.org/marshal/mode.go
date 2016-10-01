// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// ErrInvalidOctalMode is error returned if no file mode can be
// encoded or decoded.
var ErrInvalidOctalMode = errors.New("invalid file mode in octal representation")

// Mode is a helper type that can JSON encode and decode integers
// used as file mode in octal representation.
type Mode int

// MarshalJSON implements json.Marshaler interface.
// It marshals a string of octal representation of integer.
func (m Mode) MarshalJSON() ([]byte, error) {
	x, err := m.MarshalText()
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(string(x))
	if err != nil {
		return nil, ErrInvalidOctalMode
	}
	return b, err
}

// UnmarshalJSON implements json.Unamrshaler interface.
// It parses octal representations of integer.
func (m *Mode) UnmarshalJSON(data []byte) error {
	if len(data) <= 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return ErrInvalidOctalMode
	}
	if err := m.UnmarshalText(bytes.Trim(data, "\"")); err != nil {
		return ErrInvalidOctalMode
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler interface.
// It marshals a string of octal representation of integer.
func (m Mode) MarshalText() ([]byte, error) {
	if int(m) < 0 || int(m) > 0777 {
		return nil, ErrInvalidOctalMode
	}
	buf := &bytes.Buffer{}
	_, err := fmt.Fprintf(buf, "%03o", m)
	return buf.Bytes(), err
}

// UnmarshalText implements encoding.TextUnamrshaler interface.
// It parses octal representations of integer.
func (m *Mode) UnmarshalText(data []byte) error {
	i, err := strconv.ParseUint(string(data), 8, 16)
	if err != nil {
		return ErrInvalidOctalMode
	}
	*m = Mode(i)
	return nil
}

// FileMode returns os.FileMode of Mode value.
func (m Mode) FileMode() os.FileMode {
	return os.FileMode(m)
}
