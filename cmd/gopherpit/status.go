// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"resenje.org/daemon"
)

func statusCmd() {
	// Use daemon.Daemon to obtain status information and print it.
	pid, err := (&daemon.Daemon{
		PidFileName: gopherpitOptions.PidFileName,
	}).Status()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Not running:", err)
		os.Exit(2)
	}
	fmt.Println("Running: PID", pid)
}
