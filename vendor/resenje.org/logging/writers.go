// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bytes"
	"unicode"
)

// DebugLogWriter is a Logger with Write method what writes
// all messages with Debug level.
type DebugLogWriter struct {
	*Logger
}

// NewDebugLogWriter creates a writer object that will send messages
// with Debug log level to a logger.
func NewDebugLogWriter(logger *Logger) DebugLogWriter {
	return DebugLogWriter{logger}
}

// Write logs an Debug message to a logger.
func (lw DebugLogWriter) Write(p []byte) (int, error) {
	lw.Debug(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return len(p), nil
}

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
	return len(p), nil
}

// WarningLogWriter is a Logger with Write method what writes
// all messages with Warning level.
type WarningLogWriter struct {
	*Logger
}

// NewWarningLogWriter creates a writer object that will send messages
// with Warning log level to a logger.
func NewWarningLogWriter(logger *Logger) WarningLogWriter {
	return WarningLogWriter{logger}
}

// Write logs an Warning message to a logger.
func (lw WarningLogWriter) Write(p []byte) (int, error) {
	lw.Warning(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return len(p), nil
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
	return len(p), nil
}
