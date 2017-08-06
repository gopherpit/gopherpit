// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

import (
	"errors"
	"expvar"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"syscall"
	"time"

	"resenje.org/daemon"
	"resenje.org/logging"
)

// Package default variables.
var (
	DefaultFileMode          os.FileMode = 0666
	DefaultDirectoryMode     os.FileMode = 0777
	DefaultDaemonLogFileName             = "daemon.log"
)

// App provides common functionalities of an application, like
// setting a working directory and putting process in the background
// aka daemonizing and starting arbitrary functions that provide core logic.
type App struct {
	name string

	homeDir string
	logDir  string

	daemonLogFileName string
	daemonLogFileMode os.FileMode

	// A list of non-blocking or short-lived functions to be executed on
	// App.Start.
	Functions []func() error
	// A function to be executed after receiving SIGINT or SIGTERM.
	ShutdownFunc func() error
	// Instance of resenje.org/daemon.Daemon.
	Daemon *daemon.Daemon
}

// AppOptions contain optional parameters for App.
type AppOptions struct {
	// Working directory after daemonizing.
	HomeDir string
	// Directory for log files. If it is not set, logger will be configured
	// to print messages to stderr.
	LogDir string
	// File name of a PID file.
	PidFileName string
	// Mode of a PID file. Default 644.
	PidFileMode os.FileMode
	// File name in which to redirect stdout and stderr of a daemonized process.
	// If it is not set, /dev/null will be used.
	DaemonLogFileName string
	// Mode of a daemon log file. Default 644.
	DaemonLogFileMode os.FileMode
}

// NewApp creates a new instance of App, based on provided Options.
func NewApp(name string, o AppOptions) (a *App, err error) {
	a = &App{
		name:      name,
		Functions: []func() error{},
		homeDir:   o.HomeDir,
		logDir:    o.LogDir,
	}
	if o.PidFileName != "" {
		pidFileMode := o.PidFileMode
		if pidFileMode == 0 {
			pidFileMode = DefaultFileMode
		}
		a.Daemon = &daemon.Daemon{
			PidFileName: o.PidFileName,
			PidFileMode: pidFileMode,
		}
		a.daemonLogFileMode = o.DaemonLogFileMode
		if a.daemonLogFileMode == 0 {
			a.daemonLogFileMode = DefaultFileMode
		}
		a.daemonLogFileName = o.DaemonLogFileName
		if a.daemonLogFileName == "" {
			a.daemonLogFileName = DefaultDaemonLogFileName
		}
	}

	return
}

// Start executes all function in App.Functions, starts a goroutine
// that receives USR1 signal to dump debug data and blocks until INT or TERM
// signals are received.
func (a App) Start(logger *logging.Logger) error {
	// We want some fancy signal features
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGUSR1)
	go func() {
	Loop:
		for {
			sig := <-signalChannel
			logger.Noticef("received signal: %v", sig)
			switch sig {
			case syscall.SIGUSR1:
				var dir string
				if a.homeDir != "" {
					dir = filepath.Join(a.homeDir, "debug", time.Now().UTC().Format("2006-01-02_15.04.05.000000"))
					if err := os.MkdirAll(dir, DefaultDirectoryMode); err != nil {
						logger.Errorf("debug dump: create debug log dir: %s", err)
						continue Loop
					}
				}

				if dir != "" {
					f, err := os.OpenFile(filepath.Join(dir, "vars"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
					if err != nil {
						logging.Errorf("debug dump: create vars file: %s", err)
						continue
					}
					if err := expvarExport(f); err != nil {
						logging.Errorf("debug dump: write vars file: %s", err)
					}
					if err := f.Close(); err != nil {
						logging.Errorf("debug dump: close vars file: %s", err)
					}
				} else {
					expvarExport(os.Stderr)
				}

				for _, d := range []struct {
					filename   string
					profile    string
					debugLevel int
				}{
					{
						filename:   "goroutine",
						profile:    "goroutine",
						debugLevel: 2,
					},
					{
						filename:   "heap",
						profile:    "heap",
						debugLevel: 0,
					},
					{
						filename:   "heap-verbose",
						profile:    "heap",
						debugLevel: 1,
					},
					{
						filename:   "block",
						profile:    "block",
						debugLevel: 1,
					},
					{
						filename:   "threadcreate",
						profile:    "threadcreate",
						debugLevel: 1,
					},
				} {
					if dir != "" {
						f, err := os.OpenFile(filepath.Join(dir, d.filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
						if err != nil {
							logger.Errorf("debug dump: create %s dump file: %s", d.filename, err)
							continue
						}
						pprof.Lookup(d.profile).WriteTo(f, d.debugLevel)
						if err := f.Close(); err != nil {
							logger.Errorf("debug dump: close %s file: %s", d.filename, err)
						}
					} else {
						fmt.Fprintln(os.Stderr, "debug dump:", d.filename)
						pprof.Lookup(d.profile).WriteTo(os.Stderr, d.debugLevel)
					}
				}
				if dir != "" {
					logger.Infof("debug dump: %s", dir)
				} else {
					logger.Info("debug dump: done")
				}
			}
		}
	}()

	defer func() {
		// Handle panic in this goroutine
		if err := recover(); err != nil {
			// Just log the panic error and crash
			logger.Errorf("panic: %s", err)
			logger.Errorf("stack: %s", debug.Stack())
			logging.WaitForAllUnprocessedRecords()
			os.Exit(1)
		}
	}()

	logger.Info("application start")
	logger.Infof("pid %d", os.Getpid())

	// Start all async functions
	for _, function := range a.Functions {
		if err := function(); err != nil {
			return err
		}
	}

	// Wait fog termination or interrupt signals
	// We want to clean up thing at the end
	interruptChannel := make(chan os.Signal)
	signal.Notify(interruptChannel, syscall.SIGINT, syscall.SIGTERM)
	// Blocking part
	logger.Noticef("received signal: %v", <-interruptChannel)
	if a.Daemon != nil && a.Daemon.PidFileName != "" {
		// Remove Pid file only if there is a daemon
		a.Daemon.Cleanup()
	}

	if a.ShutdownFunc != nil {
		done := make(chan struct{})
		go func() {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf("shutdown panic: %s", err)
				}
			}()

			if err := a.ShutdownFunc(); err != nil {
				logger.Errorf("shutdown: %s", err)
			}
			close(done)
		}()

		// If shutdown function is blocking too long,
		// allow process termination by receiving another signal.
		interruptChannel := make(chan os.Signal)
		signal.Notify(interruptChannel, syscall.SIGINT, syscall.SIGTERM)
		// Blocking part
		select {
		case sig := <-interruptChannel:
			logger.Noticef("received signal: %v", sig)
		case <-done:
		}
	}

	logger.Info("application stop")
	// Process remaining log messages
	logging.WaitForAllUnprocessedRecords()
	return nil
}

// Daemonize puts process in the background.
func (a App) Daemonize() {
	nullFile, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var daemonFile *os.File
	if a.daemonLogFileName != "" && a.logDir != "" {
		daemonFile, err = os.OpenFile(filepath.Join(a.logDir, a.daemonLogFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, a.daemonLogFileMode)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		daemonFile = nullFile
	}

	if err := a.Daemon.Daemonize(
		a.homeDir,  // workDir
		nullFile,   // inFile
		daemonFile, // outFIle
		daemonFile, // errFile
	); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// StopDaemon send term signal to a daemonized process and reports the status.
func StopDaemon(d daemon.Daemon) error {
	err := d.Stop()
	if err == nil {
		i := 0
		for {
			if i > 10 {
				return errors.New("stop failed")
			}
			if _, err := d.Status(); err != nil {
				break
			}
			time.Sleep(250 * time.Millisecond)
			i++
		}
	}
	if err != nil {
		return fmt.Errorf("failed: %s", err)
	}
	return nil
}

func expvarExport(w io.Writer) (err error) {
	if _, err = fmt.Fprintf(w, "{\n"); err != nil {
		return
	}
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	_, err = fmt.Fprintf(w, "\n}\n")
	return
}
