// Copyright 2015 Janos Guljas. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package resenje.org/daemon provides functionality to execute binaries
// in the background. It requires no external dependencies.

package daemon

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// Daemon is a structure that holds information about PID file.
type Daemon struct {
	PidFileName string
	PidFileMode os.FileMode
}

// Daemonize terminates the execution of the initial process
// and starts a new process in the background. If workDir is not
// zero length string, os.Chdir is executed with that value.
// New process will have standard input, output and error specified by
// inFile, outFile and errFile.
// PID file will be created with the second process ID.
func (d *Daemon) Daemonize(workDir string, inFile io.Reader, outFile io.Writer, errFile io.Writer) error {
	if syscall.Getppid() != 1 {
		path, err := filepath.Abs(os.Args[0])
		if err != nil {
			return err
		}
		cmd := exec.Command(path, os.Args[1:]...)
		cmd.Stdin = inFile
		cmd.Stdout = outFile
		cmd.Stderr = errFile
		if err := cmd.Start(); err != nil {
			return err
		}
		os.Exit(0)
	}
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return err
		}
		os.Chdir(workDir)
	}

	s, err := syscall.Setsid()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(d.PidFileName, []byte(strconv.Itoa(s)), d.PidFileMode)
}

// Cleanup removes PID file.
func (d *Daemon) Cleanup() error {
	if d.PidFileName == "" {
		return nil
	}
	return os.Remove(d.PidFileName)
}

// Pid returns process ID if available.
func (d *Daemon) Pid() (int, error) {
	pid, err := ioutil.ReadFile(d.PidFileName)
	if err != nil {
		return 0, err
	}
	p, err := strconv.Atoi(string(bytes.TrimSpace(pid)))
	if err != nil {
		return 0, fmt.Errorf("%s: invalid process id", d.PidFileName)
	}
	return p, nil
}

// Process returns an os.Process and error returned by os.FindProcess
// based on the content from PID file.
func (d *Daemon) Process() (*os.Process, error) {
	p, err := d.Pid()
	if err != nil {
		return nil, err
	}
	return os.FindProcess(p)
}

// Signal sends os.Signal to the daemonized process.
func (d *Daemon) Signal(sig os.Signal) error {
	process, err := d.Process()
	if err != nil {
		return err
	}
	return process.Signal(sig)
}

// Status returns PID as int of a daemonized process. If the process is
// not running, returned error is not nil.
func (d *Daemon) Status() (pid int, err error) {
	p, err := d.Process()
	if err != nil {
		return 0, err
	}
	return p.Pid, p.Signal(syscall.Signal(0x0))
}

// Stop sends SIGTERM signal to the daemonized process. If it fails,
// SIGKILL signal is sent.
func (d *Daemon) Stop() error {
	process, err := d.Process()
	if err != nil {
		return err
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return process.Kill()
	}
	return nil
}
