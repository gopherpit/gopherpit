// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package service // import "gopherpit.com/gopherpit/pkg/service"

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/syslog"
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

var (
	defaultFileMode         os.FileMode = 0644
	defaultDirectoryMode    os.FileMode = 0755
	defaultaemonLogFileName             = "daemon.log"
)

// Service provides common functionalities of a system service, like
// setting a working directory, logging, putting process in the background
// aka daemonizing and starting arbitrary functions that provide core logic.
type Service struct {
	name string

	homeDir string
	logDir  string

	daemonLogFileName string
	daemonLogFileMode os.FileMode

	// A list of non-blocking or short-lived functions to be executed on
	// Service.Start.
	Functions []func() error
	// Instance of resenje.org/daemon.Daemon.
	Daemon *daemon.Daemon
}

// Options contain optional parameters for Service.
type Options struct {
	// Working directory of a service after daemonizing.
	HomeDir string
	// Directory for log files. If it is not set, logging will be configured
	// to print messages to stderr.
	LogDir string
	// Log files mode.
	LogFileMode os.FileMode
	// Log directories mode.
	LogDirectoryMode os.FileMode
	// LogLevel is the lowest level of log messages that will be logged.
	LogLevel logging.Level
	// Syslog facility for syslog messages. If it is not set, no logging to
	// syslog will be done.
	SyslogFacility logging.SyslogFacility
	// Syslog tag for sysylog messages.
	SyslogTag string
	// Syslog named network.
	SyslogNetwork string
	// Syslog network address.
	SyslogAddress string
	// AccessLogLevel is the lowest level of HTTP access log messages that will
	// be logged.
	AccessLogLevel logging.Level
	// Syslog facility for syslog messages of HTTP requests. If it is not set,
	// no logging to syslog will be done.
	AccessSyslogFacility logging.SyslogFacility
	// Syslog tag for sysylog messages of HTTP requests.
	AccessSyslogTag string
	// PackageAccessLogLevel is the lowest level of HTTP access log messages
	// that will be logged for package resolutions.
	PackageAccessLogLevel logging.Level
	// Syslog facility for syslog messages of package resolution requests.
	// If it is not set, no logging to syslog will be done.
	PackageAccessSyslogFacility logging.SyslogFacility
	// Syslog tag for sysylog messages of package resolution requests.
	PackageAccessSyslogTag string
	// Is logging of audit messages completely disabled.
	AuditLogDisabled bool
	// Syslog facility for syslog audit messages. If it is not set,
	// no logging to syslog will be done.
	AuditSyslogFacility logging.SyslogFacility
	// Syslog tag for sysylog audit messages.
	AuditSyslogTag string
	// If LogDir is set, but there is need to force logging to stderr,
	// ForceLogToStderr should be set to true.
	ForceLogToStderr bool
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

// NewService creates a new instance of Service, based on provided Options.
func NewService(name string, o Options) (s *Service, err error) {
	s = &Service{
		name:      name,
		Functions: []func() error{},
		homeDir:   o.HomeDir,
		logDir:    o.LogDir,
	}
	logFileMode := o.LogFileMode
	if logFileMode == 0 {
		logFileMode = defaultFileMode
	}
	logDirectoryMode := o.LogDirectoryMode
	if logDirectoryMode == 0 {
		logDirectoryMode = defaultDirectoryMode
	}
	if o.PidFileName != "" {
		pidFileMode := o.PidFileMode
		if pidFileMode == 0 {
			pidFileMode = defaultFileMode
		}
		s.Daemon = &daemon.Daemon{
			PidFileName: o.PidFileName,
			PidFileMode: pidFileMode,
		}
		s.daemonLogFileMode = o.DaemonLogFileMode
		if s.daemonLogFileMode == 0 {
			s.daemonLogFileMode = defaultFileMode
		}
		s.daemonLogFileName = o.DaemonLogFileName
		if s.daemonLogFileName == "" {
			s.daemonLogFileName = defaultaemonLogFileName
		}
	}

	// Setup logging.
	logHandlers := []logging.Handler{}
	accessLogHandlers := []logging.Handler{}
	packageAccessLogHandlers := []logging.Handler{}
	auditLogHandlers := []logging.Handler{}
	if o.LogDir == "" || o.ForceLogToStderr {
		logHandler := &logging.WriteHandler{
			Level:     o.LogLevel,
			Formatter: &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Writer:    os.Stderr,
		}
		logHandlers = append(logHandlers, logHandler)
		accessLogHandlers = append(accessLogHandlers, logHandler)
		packageAccessLogHandlers = append(packageAccessLogHandlers, logHandler)
		auditLogHandlers = append(auditLogHandlers, logHandler)
	} else {
		logHandlers = append(logHandlers, &logging.TimedFileHandler{
			Level:          o.LogLevel,
			Formatter:      &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Directory:      o.LogDir,
			FilenameLayout: "2006/01/02/" + s.name + ".log",
			FileMode:       logFileMode,
			DirectoryMode:  logDirectoryMode,
		})
		accessLogHandlers = append(accessLogHandlers, &logging.TimedFileHandler{
			Level:          o.AccessLogLevel,
			Formatter:      &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Directory:      o.LogDir,
			FilenameLayout: "2006/01/02/access.log",
			FileMode:       logFileMode,
			DirectoryMode:  logDirectoryMode,
		})
		packageAccessLogHandlers = append(packageAccessLogHandlers, &logging.TimedFileHandler{
			Level:          o.PackageAccessLogLevel,
			Formatter:      &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Directory:      o.LogDir,
			FilenameLayout: "2006/01/02/package-access.log",
			FileMode:       logFileMode,
			DirectoryMode:  logDirectoryMode,
		})
		if !o.AuditLogDisabled {
			auditLogHandlers = append(auditLogHandlers, &logging.TimedFileHandler{
				Level:          logging.DEBUG,
				Formatter:      &logging.MessageFormatter{},
				Directory:      o.LogDir,
				FilenameLayout: "2006/01/02/audit.log",
				FileMode:       logFileMode,
				DirectoryMode:  logDirectoryMode,
			})
		}
	}
	if !o.ForceLogToStderr {
		if o.SyslogFacility != "" && o.SyslogTag != "" {
			logHandlers = append(logHandlers, &logging.SyslogHandler{
				Formatter: &logging.MessageFormatter{},
				Tag:       o.SyslogTag,
				Facility:  o.SyslogFacility.Priority(),
				Severity:  syslog.Priority(o.LogLevel),
				Network:   o.SyslogNetwork,
				Address:   o.SyslogAddress,
			})
		}
		if o.AccessSyslogFacility != "" && o.AccessSyslogTag != "" {
			accessLogHandlers = append(accessLogHandlers, &logging.SyslogHandler{
				Formatter: &logging.MessageFormatter{},
				Tag:       o.AccessSyslogTag,
				Facility:  o.AccessSyslogFacility.Priority(),
				Severity:  syslog.Priority(o.AccessLogLevel),
				Network:   o.SyslogNetwork,
				Address:   o.SyslogAddress,
			})
		}
		if o.PackageAccessSyslogFacility != "" && o.PackageAccessSyslogTag != "" {
			packageAccessLogHandlers = append(packageAccessLogHandlers, &logging.SyslogHandler{
				Formatter: &logging.MessageFormatter{},
				Tag:       o.PackageAccessSyslogTag,
				Facility:  o.PackageAccessSyslogFacility.Priority(),
				Severity:  syslog.Priority(o.PackageAccessLogLevel),
				Network:   o.SyslogNetwork,
				Address:   o.SyslogAddress,
			})
		}
		if o.AuditSyslogFacility != "" && o.AuditSyslogTag != "" && !o.AuditLogDisabled {
			auditLogHandlers = append(auditLogHandlers, &logging.SyslogHandler{
				Formatter: &logging.MessageFormatter{},
				Tag:       o.AuditSyslogTag,
				Facility:  o.AuditSyslogFacility.Priority(),
				Severity:  syslog.LOG_DEBUG,
				Network:   o.SyslogNetwork,
				Address:   o.SyslogAddress,
			})
		}
	}

	logging.RemoveLoggers()
	logger, err := logging.NewLogger("default", logging.DEBUG, logHandlers, 0)
	if err != nil {
		err = fmt.Errorf("default logger error: %s", err)
		return
	}
	log.SetOutput(logging.NewInfoLogWriter(logger))
	log.SetFlags(0)
	if _, err = logging.NewLogger("access", logging.DEBUG, accessLogHandlers, 0); err != nil {
		err = fmt.Errorf("access logger error: %s", err)
		return
	}
	if _, err = logging.NewLogger("package-access", logging.DEBUG, packageAccessLogHandlers, 0); err != nil {
		err = fmt.Errorf("package access logger error: %s", err)
		return
	}
	if _, err = logging.NewLogger("audit", logging.DEBUG, auditLogHandlers, 0); err != nil {
		err = fmt.Errorf("audit logger error: %s", err)
		return
	}
	return
}

// Start executes all function in Service.Functions, starts a goroutine
// that receives USR1 signal to dump debug data and blocks until INT or TERM
// signals are received.
func (s Service) Start() error {
	// We want some fancy signal features
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
	Loop:
		for {
			sig := <-signalChannel
			logging.Noticef("received signal: %v", sig)
			switch sig {
			case syscall.SIGUSR1:
				var dir string
				if s.homeDir != "" {
					dir = filepath.Join(s.homeDir, "debug", time.Now().UTC().Format("2006-01-02_15.04.05.000000"))
					if err := os.MkdirAll(dir, defaultDirectoryMode); err != nil {
						logging.Errorf("debug dump: create debug log dir: %s", err)
						continue Loop
					}
				}

				info, err := json.MarshalIndent(NewInfo(), "", "    ")
				if err != nil {
					logging.Errorf("debug dump: decode service info: %s", err)
				}
				if dir != "" {
					f, err := os.OpenFile(filepath.Join(dir, "info"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultFileMode)
					if err != nil {
						logging.Errorf("debug dump: create info dump file: %s", err)
						continue
					}
					f.Write(info)
					if err := f.Close(); err != nil {
						logging.Errorf("debug dump: close service info file: %s", err)
					}
				} else {
					fmt.Fprintln(os.Stderr, string(info))
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
						f, err := os.OpenFile(filepath.Join(dir, d.filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultFileMode)
						if err != nil {
							logging.Errorf("debug dump: create %s dump file: %s", d.filename, err)
							continue
						}
						pprof.Lookup(d.profile).WriteTo(f, d.debugLevel)
						if err := f.Close(); err != nil {
							logging.Errorf("debug dump: close %s file: %s", d.filename, err)
						}
					} else {
						fmt.Fprintln(os.Stderr, "debug dump:", d.filename)
						pprof.Lookup(d.profile).WriteTo(os.Stderr, d.debugLevel)
					}
				}
				if dir != "" {
					logging.Infof("debug dump: %s", dir)
				} else {
					logging.Info("debug dump: done")
				}
			}
		}
	}()

	defer func() {
		// Handle panic in this goroutine
		if err := recover(); err != nil {
			// Just log the panic error and crash
			logging.Errorf("panic: %s", err)
			logging.Errorf("stack: %s", debug.Stack())
			logging.WaitForAllUnprocessedRecords()
			os.Exit(1)
		}
	}()

	logging.Info("Service start")
	logging.Infof("Pid %d", os.Getpid())

	// Start all async functions
	for _, function := range s.Functions {
		if err := function(); err != nil {
			return err
		}
	}

	// Wait fog termination or interrup signals
	// We want to clean up thing at the end
	interruptChannel := make(chan os.Signal)
	signal.Notify(interruptChannel, syscall.SIGINT, syscall.SIGTERM)
	// Blocking part
	logging.Noticef("Received signal: %v", <-interruptChannel)
	if s.Daemon != nil && s.Daemon.PidFileName != "" {
		// Remove Pid file only if there is a daemon
		s.Daemon.Cleanup()
	}

	logging.Info("Service stop")
	// Process remaining log messages
	logging.WaitForAllUnprocessedRecords()
	return nil
}

// Daemonize puts process in the background.
func (s Service) Daemonize() {
	nullFile, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var daemonFile *os.File
	if s.daemonLogFileName != "" && s.logDir != "" {
		daemonFile, err = os.OpenFile(filepath.Join(s.logDir, s.daemonLogFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, s.daemonLogFileMode)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		daemonFile = nullFile
	}

	if err := s.Daemon.Daemonize(
		s.homeDir,  // workDir
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
