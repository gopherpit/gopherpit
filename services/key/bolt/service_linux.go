// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package boltKey

import "syscall"

func init() {
	mmapFlags = syscall.MAP_POPULATE
}
