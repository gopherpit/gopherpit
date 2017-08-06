// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

import (
	"os"
	"path/filepath"

	"resenje.org/logging"
	"resenje.org/marshal"
)

// LoggingOptions defines parameters related to service's core functionality.
type LoggingOptions struct {
	name                 string
	LogDir               string                 `json:"log-dir" yaml:"log-dir" envconfig:"LOG_DIR"`
	LogLevel             logging.Level          `json:"log-level" yaml:"log-level" envconfig:"LOG_LEVEL"`
	SyslogFacility       logging.SyslogFacility `json:"syslog-facility" yaml:"syslog-facility" envconfig:"SYSLOG_FACILITY"`
	SyslogTag            string                 `json:"syslog-tag" yaml:"syslog-tag" envconfig:"SYSLOG_TAG"`
	SyslogNetwork        string                 `json:"syslog-network" yaml:"syslog-network" envconfig:"SYSLOG_NETWORK"`
	SyslogAddress        string                 `json:"syslog-address" yaml:"syslog-address" envconfig:"SYSLOG_ADDRESS"`
	AccessLogLevel       logging.Level          `json:"access-log-level" yaml:"access-log-level" envconfig:"ACCESS_LOG_LEVEL"`
	AccessSyslogFacility logging.SyslogFacility `json:"access-syslog-facility" yaml:"access-syslog-facility" envconfig:"ACCESS_SYSLOG_FACILITY"`
	AccessSyslogTag      string                 `json:"access-syslog-tag" yaml:"access-syslog-tag" envconfig:"ACCESS_SYSLOG_TAG"`
	AuditLogLevel        logging.Level          `json:"audit-log-level" yaml:"audit-log-level" envconfig:"AUDIT_LOG_LEVEL"`
	AuditSyslogFacility  logging.SyslogFacility `json:"audit-syslog-facility" yaml:"audit-syslog-facility" envconfig:"AUDIT_SYSLOG_FACILITY"`
	AuditSyslogTag       string                 `json:"audit-syslog-tag" yaml:"audit-syslog-tag" envconfig:"AUDIT_SYSLOG_TAG"`
	DaemonLogFileName    string                 `json:"daemon-log-file" yaml:"daemon-log-file" envconfig:"DAEMON_LOG_FILE"`
	DaemonLogFileMode    marshal.Mode           `json:"daemon-log-file-mode" yaml:"daemon-log-file-mode" envconfig:"DAEMON_LOG_FILE_MODE"`
}

// NewLoggingOptions initializes LoggingOptions with default values.
func NewLoggingOptions(name, baseDir string) *LoggingOptions {
	return &LoggingOptions{
		name:                 name,
		LogDir:               filepath.Join(baseDir, "log"),
		LogLevel:             logging.DEBUG,
		SyslogFacility:       "",
		SyslogTag:            name,
		SyslogNetwork:        "",
		SyslogAddress:        "",
		AccessLogLevel:       logging.DEBUG,
		AccessSyslogFacility: "",
		AccessSyslogTag:      name + "-access",
		AuditLogLevel:        logging.DEBUG,
		AuditSyslogFacility:  "",
		AuditSyslogTag:       name + "-audit",
		DaemonLogFileName:    "daemon.log",
		DaemonLogFileMode:    0644,
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *LoggingOptions) VerifyAndPrepare() error {
	for _, dir := range []string{
		o.LogDir,
	} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0777); err != nil {
				return err
			}
		}
	}
	return nil
}
