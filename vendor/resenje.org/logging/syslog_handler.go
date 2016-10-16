// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

package logging

import (
	"log/syslog"
)

// SyslogHandler sends all messages to syslog.
type SyslogHandler struct {
	NullHandler

	Formatter Formatter
	Tag       string
	Facility  syslog.Priority
	Severity  syslog.Priority
	// Network is a named network to connect to syslog.
	// Known networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only),
	// "udp", "udp4" (IPv4-only), "udp6" (IPv6-only), "ip", "ip4" (IPv4-only),
	// "ip6" (IPv6-only), "unix", "unixgram" and "unixpacket".
	// If network is empty, SyslogHandler will connect to the local
	// syslog server.
	Network string
	// Address is a network address to connect to syslog.
	Address string
	writter *syslog.Writer
}

// Handle sends a record message to syslog Writer.
func (handler *SyslogHandler) Handle(record *Record) error {

	if handler.writter == nil {
		writter, err := syslog.Dial(
			handler.Network,
			handler.Address,
			handler.Facility|handler.Severity,
			handler.Tag,
		)

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
