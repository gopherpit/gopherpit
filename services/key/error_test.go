// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import "testing"

func TestNewError(t *testing.T) {
	var errorCode = 1
	err := NewError(errorCode, "error 1")
	if err != ErrorRegistry.Error(errorCode) {
		t.Errorf("New error was not found in ErrorRegistry")
	}
}

func TestNewErrorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Adding new error with existing code did not panic")
		}
	}()

	NewError(1, "error 1")
	NewError(1, "error 2")
}
