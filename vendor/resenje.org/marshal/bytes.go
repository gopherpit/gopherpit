// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"unicode"
)

type byteSize float64

const (
	aByte byteSize = 1 << (iota * 10)
	kiloByte
	megaByte
	gigaByte
	teraByte
	petaByte
	exaByte
	zettaByte
	yottaByte
)

var byteSizes = map[string]byteSize{
	"b":  aByte,
	"kb": kiloByte,
	"mb": megaByte,
	"gb": gigaByte,
	"tb": teraByte,
	"pb": petaByte,
	"eb": exaByte,
	"zb": zettaByte,
	"yb": yottaByte,
}

var byteSuffixes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// Bytes is a helper type that represents length of data in bytes.
type Bytes float64

// MarshalJSON implements json.Marshaler interface.
// It marshals a string representation of length of bytes.
func (b Bytes) MarshalJSON() ([]byte, error) {
	x, err := b.MarshalText()
	if err != nil {
		return nil, err
	}
	return append([]byte{'"'}, append(x, '"')...), nil
}

// UnmarshalJSON implements json.Unamrshaler interface.
// It parses length of bytes string.
func (b *Bytes) UnmarshalJSON(data []byte) error {
	return b.UnmarshalText(bytes.Trim(data, "\""))
}

// MarshalText implements encoding.TextMarshaler interface.
// It marshals a string representation of length of bytes.
func (b Bytes) MarshalText() ([]byte, error) {
	buf := &bytes.Buffer{}
	if b < 10 {
		_, err := fmt.Fprintf(buf, "%dB", int(b))
		return buf.Bytes(), err
	}
	e := math.Floor(math.Log(float64(b)) / math.Log(1024))
	suffix := byteSuffixes[int(e)]
	val := math.Floor(float64(b)/math.Pow(1024, e)*10+0.5) / 10
	f := "%.0f%s"
	if val < 10 {
		f = "%.1f%s"
	}

	_, err := fmt.Fprintf(buf, f, val, suffix)
	return buf.Bytes(), err
}

// UnmarshalText implements encoding.TextUnamrshaler interface.
// It parses length of bytes string.
func (b *Bytes) UnmarshalText(data []byte) error {
	lastDigit := 0
	for _, r := range string(data) {
		if !(unicode.IsDigit(r) || r == '.') {
			break
		}
		lastDigit++
	}

	f, err := strconv.ParseFloat(string(data[:lastDigit]), 64)
	if err != nil {
		return err
	}

	extra := bytes.ToLower(bytes.TrimSpace(data[lastDigit:]))
	if m, ok := byteSizes[string(extra)]; ok {
		f *= float64(m)
		if f >= math.MaxFloat64 {
			return fmt.Errorf("too large: %s", data)
		}
		*b = Bytes(f)
		return nil
	}

	return fmt.Errorf("unhandled size name: %s", extra)
}

// Bytes returns an integer value of length of bytes.
func (b Bytes) Bytes() float64 {
	return float64(b)
}
