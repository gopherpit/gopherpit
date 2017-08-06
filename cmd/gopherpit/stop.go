// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"resenje.org/daemon"

	"resenje.org/x/application"
)

func stopCmd() {
	err := application.StopDaemon(daemon.Daemon{
		PidFileName: options.PidFileName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
	fmt.Println("Stopped")
}
