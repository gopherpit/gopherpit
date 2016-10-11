// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dataDump

import (
	"io"
	"time"
)

// Interface defines method to retrieve data Dump. If ifModifiedSince
// is not nil and data is not changed since provided time,
// both return values, Dump and error, will be nil.
type Interface interface {
	DataDump(ifModifiedSince *time.Time) (dump *Dump, err error)
}

// Dump defines a structure that holds dump metadata and body as reader interface.
// Body must be closed after the read is done.
type Dump struct {
	Name        string
	ContentType string
	Length      int64
	ModTime     *time.Time
	Body        io.ReadCloser
}
