// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package logging

// SyslogHander is not available on Windows platform.
type SyslogHandler struct {
	NullHandler
}
