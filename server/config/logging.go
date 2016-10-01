// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"resenje.org/logging"
	"resenje.org/marshal"
)

// LoggingOptions defines parameters related to service's core functionality.
type LoggingOptions struct {
	LogDir               string                 `json:"log-dir" envconfig:"LOG_DIR"`
	LogLevel             logging.Level          `json:"log-level" envconfig:"LOG_LEVEL"`
	LogFileMode          marshal.Mode           `json:"log-file-mode" envconfig:"LOG_FILE_MODE"`
	LogDirectoryMode     marshal.Mode           `json:"log-directory-mode" envconfig:"LOG_DIRECTORY_MODE"`
	SyslogFacility       logging.SyslogFacility `json:"syslog-facility" envconfig:"SYSLOG_FACILITY"`
	SyslogTag            string                 `json:"syslog-tag" envconfig:"SYSLOG_TAG"`
	AccessLogLevel       logging.Level          `json:"access-log-level" envconfig:"ACCESS_LOG_LEVEL"`
	AccessSyslogFacility logging.SyslogFacility `json:"access-syslog-facility" envconfig:"ACCESS_SYSLOG_FACILITY"`
	AccessSyslogTag      string                 `json:"access-syslog-tag" envconfig:"ACCESS_SYSLOG_TAG"`
	AuditLogDisabled     bool                   `json:"audit-log-disabled" envconfig:"AUDIT_LOG_DISABLED"`
	AuditSyslogFacility  logging.SyslogFacility `json:"audit-syslog-facility" envconfig:"AUDIT_SYSLOG_FACILITY"`
	AuditSyslogTag       string                 `json:"audit-syslog-tag" envconfig:"AUDIT_SYSLOG_TAG"`
	DaemonLogFileName    string                 `json:"daemon-log-file" envconfig:"DAEMON_LOG_FILE"`
	DaemonLogFileMode    marshal.Mode           `json:"daemon-log-file-mode" envconfig:"DAEMON_LOG_FILE_MODE"`
}

// NewLoggingOptions initializes LoggingOptions with default values.
func NewLoggingOptions() *LoggingOptions {
	return &LoggingOptions{
		LogDir:               filepath.Join(BaseDir, "log"),
		LogLevel:             logging.DEBUG,
		LogFileMode:          0644,
		LogDirectoryMode:     0755,
		SyslogFacility:       "",
		SyslogTag:            Name,
		AccessLogLevel:       logging.DEBUG,
		AccessSyslogFacility: "",
		AuditLogDisabled:     false,
		AccessSyslogTag:      Name + "-access",
		AuditSyslogFacility:  "",
		AuditSyslogTag:       Name + "-audit",
		DaemonLogFileName:    "daemon.log",
		DaemonLogFileMode:    0644,
	}
}

// Update updates options by loading logging.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *LoggingOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "logging.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_logging", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// Verify checks if configuration values are valid and if all requirements are
// set for service to start.
func (o *LoggingOptions) Verify() (help string, err error) {
	return
}

// String returns a JSON representation of the options.
func (o *LoggingOptions) String() string {
	data, _ := json.MarshalIndent(o, "", "    ")
	return string(data)
}

// Prepare creates configured directories for home, storage, logs and
// temporary files.
func (o *LoggingOptions) Prepare() error {
	for _, dir := range []string{
		o.LogDir,
	} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}
