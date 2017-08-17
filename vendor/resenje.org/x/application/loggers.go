// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

import (
	"io"
	"log"
	"log/syslog"

	"resenje.org/logging"
)

// Loggers holds common configuration for a set of loggers.
type Loggers struct {
	writer io.Writer
}

// LoggersOption sets configuration parameters for Loggers
type LoggersOption func(*Loggers)

// WithForcedWriter overrides all handlers with the
// Write logging handler with provied writer.
func WithForcedWriter(w io.Writer) LoggersOption {
	return func(l *Loggers) {
		l.writer = w
	}
}

// NewLoggers creates a new instalce of Loggers
// with provided options.
func NewLoggers(opts ...LoggersOption) *Loggers {
	l := &Loggers{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// NewLogger add a new named logger with level and handlers.
func (l Loggers) NewLogger(name string, level logging.Level, handlers ...logging.Handler) *logging.Logger {
	hs := []logging.Handler{}
	if l.writer != nil {
		hs = append(hs, &logging.WriteHandler{
			Formatter: &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Writer:    l.writer,
			Level:     logging.DEBUG,
		})
	} else {
		for _, h := range handlers {
			if h != nil {
				hs = append(hs, h)
			}
		}
	}
	return logging.NewLogger(name, level, hs, 0)
}

// NewSyslogHandler is a helper to easily create
// logging.SyslogHandler.
func NewSyslogHandler(facility logging.SyslogFacility, tag, network, address string) logging.Handler {
	if facility != "" {
		return &logging.SyslogHandler{
			Formatter: &logging.MessageFormatter{},
			Tag:       tag,
			Facility:  facility.Priority(),
			Severity:  syslog.Priority(logging.DEBUG),
			Network:   network,
			Address:   address,
		}
	}
	return nil
}

// NewTimedFileHandler is a helper to easily create
// logging.TimedFileHandler.
func NewTimedFileHandler(dir, tag string) logging.Handler {
	if dir != "" {
		return &logging.TimedFileHandler{
			Formatter:      &logging.StandardFormatter{TimeFormat: logging.StandardTimeFormat},
			Directory:      dir,
			FilenameLayout: "2006/01/02",
			FileExtension:  "/" + tag + ".log",
			Level:          logging.DEBUG,
		}
	}
	return nil
}

// SetStdLogger sets the output of stdlib's log logger
// to the "default" logger.
func SetStdLogger() {
	logger, _ := logging.GetLogger("default")
	log.SetOutput(logging.NewInfoLogWriter(logger))
	log.SetFlags(0)
}
