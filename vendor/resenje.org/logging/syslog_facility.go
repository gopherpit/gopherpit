// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"log/syslog"
	"strings"
)

// SyslogFacility is a string representation of syslog facility.
type SyslogFacility string

var syslogPriorities = map[string]syslog.Priority{
	"kern":     syslog.LOG_KERN,
	"user":     syslog.LOG_USER,
	"mail":     syslog.LOG_MAIL,
	"daemon":   syslog.LOG_DAEMON,
	"auth":     syslog.LOG_AUTH,
	"syslog":   syslog.LOG_SYSLOG,
	"lpr":      syslog.LOG_LPR,
	"news":     syslog.LOG_NEWS,
	"uucp":     syslog.LOG_UUCP,
	"cron":     syslog.LOG_CRON,
	"authpriv": syslog.LOG_AUTHPRIV,
	"ftp":      syslog.LOG_FTP,

	"local0": syslog.LOG_LOCAL0,
	"local1": syslog.LOG_LOCAL1,
	"local2": syslog.LOG_LOCAL2,
	"local3": syslog.LOG_LOCAL3,
	"local4": syslog.LOG_LOCAL4,
	"local5": syslog.LOG_LOCAL5,
	"local6": syslog.LOG_LOCAL6,
	"local7": syslog.LOG_LOCAL7,
}

// String returns a string representation of SyslogFacility.
func (s SyslogFacility) String() string {
	return string(s)
}

// Priority returns a syslog.Priority representation of SyslogFacility.
func (s SyslogFacility) Priority() syslog.Priority {
	return syslogPriorities[strings.ToLower(s.String())]
}

// OK checks if SyslogFacility is valid.
func (s SyslogFacility) OK() (ok bool) {
	_, ok = syslogPriorities[strings.ToLower(s.String())]
	return
}
