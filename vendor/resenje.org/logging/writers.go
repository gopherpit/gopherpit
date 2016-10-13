// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bytes"
	"unicode"
)

// InfoLogWriter is a Logger with Write method what writes
// all messages with Info level.
type InfoLogWriter struct {
	*Logger
}

// NewInfoLogWriter creates a writer object that will send messages
// with Info log level to a logger.
func NewInfoLogWriter(logger *Logger) InfoLogWriter {
	return InfoLogWriter{logger}
}

// Write logs an Info message to a logger.
func (lw InfoLogWriter) Write(p []byte) (int, error) {
	lw.Info(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return 0, nil
}

// ErrorLogWriter is a Logger with Write method what writes
// all messages with Error level.
type ErrorLogWriter struct {
	*Logger
}

// NewErrorLogWriter creates a writer object that will send messages
// with Error log level to a logger.
func NewErrorLogWriter(logger *Logger) ErrorLogWriter {
	return ErrorLogWriter{logger}
}

// Write logs an Error message to a logger.
func (lw ErrorLogWriter) Write(p []byte) (int, error) {
	lw.Error(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return 0, nil
}
