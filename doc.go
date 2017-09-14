// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gopherpit is root package of the project.
package gopherpit // import "gopherpit.com/gopherpit"

//go:generate go-bindata -prefix assets -o server/data/assets/data.go -pkg assets -nomemcopy -nocompress assets/...
//go:generate go-bindata -prefix static -o server/data/static/data.go -pkg static -nomemcopy -nocompress static/...
//go:generate go-bindata -prefix templates -o server/data/templates/data.go -pkg templates -nomemcopy -nocompress templates/...
