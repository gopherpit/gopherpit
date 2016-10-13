// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

package logging

import (
	"log/syslog"
)

// SyslogHandler sends all messages to syslog daemon.
type SyslogHandler struct {
	NullHandler

	Formatter Formatter
	Tag       string
	Facility  syslog.Priority
	Severity  syslog.Priority
	writter   *syslog.Writer
}

// Handle sends a record message to syslog Writer.
func (handler *SyslogHandler) Handle(record *Record) error {

	if handler.writter == nil {
		writter, err := syslog.New(handler.Facility|handler.Severity, handler.Tag)

		if err != nil {
			return err
		}
		handler.writter = writter
	}

	msg := handler.Formatter.Format(record)

	switch record.Level {
	case EMERGENCY:
		return handler.writter.Emerg(msg)
	case ALERT:
		return handler.writter.Alert(msg)
	case CRITICAL:
		return handler.writter.Crit(msg)
	case ERROR:
		return handler.writter.Err(msg)
	case WARNING:
		return handler.writter.Warning(msg)
	case NOTICE:
		return handler.writter.Notice(msg)
	case INFO:
		return handler.writter.Info(msg)
	default:
		return handler.writter.Debug(msg)
	}
}

// Close closes an associated syslog Writer.
func (handler *SyslogHandler) Close() error {
	if handler.writter == nil {
		return nil
	}
	return handler.writter.Close()
}

// GetLevel returns a Level from handler's Severity.
func (handler *SyslogHandler) GetLevel() Level {
	return Level(handler.Severity)
}
