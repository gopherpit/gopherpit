// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
)

var (
	usage = `USAGE

  %s [options...] [command]

  Executing the program without specifying a command will start a process in
  the foreground and log all messages to stderr.

COMMANDS

  daemon
    Start program in the background.

  stop
    Stop program that runs in the background.

  status
    Dispaly status of a running process.

  config
    Print configuration that program will load on start. This command is
    dependent of -config-dir option value.

  debug-dump
    Send to a running process USR1 signal to log debug information in the log.

  version
    Print version to Stdout.

OPTIONS

`

	helpFooter = `
COPYRIGHT

  Copyright (C) 2016,2017 Janoš Guljaš <janos@resenje.org>

  All rights reserved.
  Use of this source code is governed by a BSD-style
  license that can be found in the LICENSE file.
`
)

func helpCmd() {
	cli.Usage()
	fmt.Fprintln(os.Stderr, helpFooter)
}

func helpUnknownCmd(cmd string) {
	fmt.Fprintln(os.Stderr, "unknown command:", cmd)
	cli.Usage()
	os.Exit(2)
}
