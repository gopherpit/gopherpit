// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
	"resenje.org/logging"
	"resenje.org/marshal"
)

// LoggingOptions defines parameters related to service's core functionality.
type LoggingOptions struct {
	LogDir                      string                 `json:"log-dir" yaml:"log-dir" envconfig:"LOG_DIR"`
	LogLevel                    logging.Level          `json:"log-level" yaml:"log-level" envconfig:"LOG_LEVEL"`
	LogFileMode                 marshal.Mode           `json:"log-file-mode" yaml:"log-file-mode" envconfig:"LOG_FILE_MODE"`
	LogDirectoryMode            marshal.Mode           `json:"log-directory-mode" yaml:"log-directory-mode" envconfig:"LOG_DIRECTORY_MODE"`
	SyslogFacility              logging.SyslogFacility `json:"syslog-facility" yaml:"syslog-facility" envconfig:"SYSLOG_FACILITY"`
	SyslogTag                   string                 `json:"syslog-tag" yaml:"syslog-tag" envconfig:"SYSLOG_TAG"`
	SyslogNetwork               string                 `json:"syslog-network" yaml:"syslog-network" envconfig:"SYSLOG_NETWORK"`
	SyslogAddress               string                 `json:"syslog-address" yaml:"syslog-address" envconfig:"SYSLOG_ADDRESS"`
	AccessLogLevel              logging.Level          `json:"access-log-level" yaml:"access-log-level" envconfig:"ACCESS_LOG_LEVEL"`
	AccessSyslogFacility        logging.SyslogFacility `json:"access-syslog-facility" yaml:"access-syslog-facility" envconfig:"ACCESS_SYSLOG_FACILITY"`
	AccessSyslogTag             string                 `json:"access-syslog-tag" yaml:"access-syslog-tag" envconfig:"ACCESS_SYSLOG_TAG"`
	PackageAccessLogLevel       logging.Level          `json:"package-access-log-level" yaml:"package-access-log-level" envconfig:"PACKAGE_ACCESS_LOG_LEVEL"`
	PackageAccessSyslogFacility logging.SyslogFacility `json:"package-access-syslog-facility" yaml:"package-access-syslog-facility" envconfig:"PACKAGE_ACCESS_SYSLOG_FACILITY"`
	PackageAccessSyslogTag      string                 `json:"package-access-syslog-tag" yaml:"package-access-syslog-tag" envconfig:"PACKAGE_ACCESS_SYSLOG_TAG"`
	AuditLogDisabled            bool                   `json:"audit-log-disabled" yaml:"audit-log-disabled" envconfig:"AUDIT_LOG_DISABLED"`
	AuditSyslogFacility         logging.SyslogFacility `json:"audit-syslog-facility" yaml:"audit-syslog-facility" envconfig:"AUDIT_SYSLOG_FACILITY"`
	AuditSyslogTag              string                 `json:"audit-syslog-tag" yaml:"audit-syslog-tag" envconfig:"AUDIT_SYSLOG_TAG"`
	DaemonLogFileName           string                 `json:"daemon-log-file" yaml:"daemon-log-file" envconfig:"DAEMON_LOG_FILE"`
	DaemonLogFileMode           marshal.Mode           `json:"daemon-log-file-mode" yaml:"daemon-log-file-mode" envconfig:"DAEMON_LOG_FILE_MODE"`
}

// NewLoggingOptions initializes LoggingOptions with default values.
func NewLoggingOptions() *LoggingOptions {
	return &LoggingOptions{
		LogDir:                      filepath.Join(BaseDir, "log"),
		LogLevel:                    logging.DEBUG,
		LogFileMode:                 0644,
		LogDirectoryMode:            0755,
		SyslogFacility:              "",
		SyslogTag:                   Name,
		SyslogNetwork:               "",
		SyslogAddress:               "",
		AccessLogLevel:              logging.DEBUG,
		AccessSyslogFacility:        "",
		AccessSyslogTag:             Name + "-access",
		PackageAccessLogLevel:       logging.DEBUG,
		PackageAccessSyslogFacility: "",
		PackageAccessSyslogTag:      Name + "-package-access",
		AuditLogDisabled:            false,
		AuditSyslogFacility:         "",
		AuditSyslogTag:              Name + "-audit",
		DaemonLogFileName:           "daemon.log",
		DaemonLogFileMode:           0644,
	}
}

// Update updates options by loading logging.json files.
func (o *LoggingOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "logging.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "logging.json")
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
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Prepare creates configured directories for home, storage, logs and
// temporary files.
func (o *LoggingOptions) Prepare() error {
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
