// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import "bytes"

var (
	checkboxTrue  = []byte(`"on"`)
	checkboxFalse = []byte(`false`)
)

// Checkbox is a helper type to unmarshal value from HTML checkbox
// into boolean.
type Checkbox bool

// MarshalJSON returns the JSON encoding of Checkbox.
//  - "on" if it is checked
//  - false if it is not checked
func (c Checkbox) MarshalJSON() ([]byte, error) {
	if c == true {
		return checkboxTrue, nil
	}
	return checkboxFalse, nil
}

// UnmarshalJSON parses the JSON-encoded data and sets Checkbox to true or false.
func (c *Checkbox) UnmarshalJSON(data []byte) error {
	*c = false
	if bytes.Equal(data, checkboxTrue) || bytes.Equal(data, []byte("true")) {
		*c = true
	}
	return nil
}

// Bool returns boolean value of a Checkbox type.
func (c Checkbox) Bool() bool {
	return bool(c)
}
