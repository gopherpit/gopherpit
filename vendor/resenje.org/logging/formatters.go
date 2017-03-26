// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// StandardTimeFormat defines a representation of timestamps in log lines.
	StandardTimeFormat = "2006-01-02 15:04:05.000Z07:00"
	tolerance          = 100 * time.Millisecond
)

// Formatter is interface for defining new custom log messages formats.
type Formatter interface {
	// Should return string that represents log message based on provided Record.
	Format(record *Record) string
}

// StandardFormatter adds time of logging with message to be logged.
type StandardFormatter struct {
	TimeFormat string
}

// Format constructs string for logging. Time of logging is added to log message.
// Also, if time of logging is more then 100 miliseconds in the passt, both
// times will be added to message (time when application sent log and time when
// message was processed). Otherwise, only time of processing will be written.
func (formatter *StandardFormatter) Format(record *Record) string {
	var message string
	now := time.Now()
	if now.Sub(record.Time) <= tolerance {
		message = record.Message
	} else {
		message = fmt.Sprintf("[%v] %v", record.Time.Format(formatter.TimeFormat), record.Message)
	}
	return fmt.Sprintf("[%v] %v %v", now.Format(formatter.TimeFormat), record.Level, message)
}

// JSONFormatter creates JSON struct with provided record.
type JSONFormatter struct{}

// Format creates JSON struct from provided record and returns it.
func (formatter *JSONFormatter) Format(record *Record) string {
	data, _ := json.Marshal(record)
	return string(data)
}

// MessageFormatter logs only Message from the Record.
type MessageFormatter struct{}

// Format returns only the Record Message
func (formatter *MessageFormatter) Format(record *Record) string {
	return record.Message
}
