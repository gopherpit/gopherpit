// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"os"
)

var defaultLogger *Logger

// newDefaultLogger creates new logger that writes log messages to
// standard output in DEBUG level. This logger is automatically created and
// this function needs to be called only if all loggers were removed
// and default logger needs recreating.
func newDefaultLogger() *Logger {
	return NewLogger("default", DEBUG, []Handler{
		&WriteHandler{
			Level:     DEBUG,
			Formatter: &StandardFormatter{TimeFormat: StandardTimeFormat},
			Writer:    os.Stderr,
		},
	}, 0)
}

func getDefaultLogger() (l *Logger) {
	if defaultLogger == nil {
		defaultLogger = newDefaultLogger()
	}
	return defaultLogger
}

// Pause temporarily stops processing of log messages in default logger.
func Pause() {
	getDefaultLogger().Pause()
}

// Unpause continues processing of log messages in default logger.
func Unpause() {
	getDefaultLogger().Unpause()
}

// Stop permanently stops processing of log messages in default logger.
func Stop() {
	getDefaultLogger().Stop()
}

// SetLevel sets maximum level for log messages that default logger
// will process.
func SetLevel(level Level) {
	getDefaultLogger().SetLevel(level)
}

// SetBufferLength sets default buffer length for default logger.
func SetBufferLength(length int) {
	getDefaultLogger().SetBufferLength(length)
}

// AddHandler adds new handler to default logger.
func AddHandler(handler Handler) {
	getDefaultLogger().AddHandler(handler)
}

// ClearHandlers remove all handlers from default logger.
func ClearHandlers() {
	getDefaultLogger().ClearHandlers()
}

// Logf logs provided message with formatting with default logger.
func Logf(level Level, format string, a ...interface{}) {
	getDefaultLogger().log(level, format, a...)
}

// Log logs provided message with default logger.
func Log(level Level, a ...interface{}) {
	getDefaultLogger().log(level, "", a...)
}

// Emergencyf logs provided message with formatting in EMERGENCY level
// with default logger.
func Emergencyf(format string, a ...interface{}) {
	getDefaultLogger().log(EMERGENCY, format, a...)
}

// Emergency logs provided message in EMERGENCY level with default logger.
func Emergency(a ...interface{}) {
	getDefaultLogger().log(EMERGENCY, "", a...)
}

// Alertf logs provided message with formatting in ALERT level
// with default logger.
func Alertf(format string, a ...interface{}) {
	getDefaultLogger().log(ALERT, format, a...)
}

// Alert logs provided message in ALERT level with default logger.
func Alert(a ...interface{}) {
	getDefaultLogger().log(ALERT, "", a...)
}

// Criticalf logs provided message with formatting in CRITICAL level
// with default logger.
func Criticalf(format string, a ...interface{}) {
	getDefaultLogger().log(CRITICAL, format, a...)
}

// Critical logs provided message in CRITICAL level with default logger.
func Critical(a ...interface{}) {
	getDefaultLogger().log(CRITICAL, "", a...)
}

// Errorf logs provided message with formatting in ERROR level
// with default logger.
func Errorf(format string, a ...interface{}) {
	getDefaultLogger().log(ERROR, format, a...)
}

// Error logs provided message in ERROR level with default logger.
func Error(a ...interface{}) {
	getDefaultLogger().log(ERROR, "", a...)
}

// Warningf logs provided message with formatting in WARNING level
// with default logger.
func Warningf(format string, a ...interface{}) {
	getDefaultLogger().log(WARNING, format, a...)
}

// Warning logs provided message in WARNING level with default logger.
func Warning(a ...interface{}) {
	getDefaultLogger().log(WARNING, "", a...)
}

// Noticef logs provided message with formatting in NOTICE level
// with default logger.
func Noticef(format string, a ...interface{}) {
	getDefaultLogger().log(NOTICE, format, a...)
}

// Notice logs provided message in NOTICE level with default logger.
func Notice(a ...interface{}) {
	getDefaultLogger().log(NOTICE, "", a...)
}

// Infof logs provided message with formatting in INFO level
// with default logger.
func Infof(format string, a ...interface{}) {
	getDefaultLogger().log(INFO, format, a...)
}

// Info logs provided message in INFO level with default logger.
func Info(a ...interface{}) {
	getDefaultLogger().log(INFO, "", a...)
}

// Debugf logs provided message with formatting in DEBUG level
// with default logger.
func Debugf(format string, a ...interface{}) {
	getDefaultLogger().log(DEBUG, format, a...)
}

// Debug logs provided message in DEBUG level with default logger.
func Debug(a ...interface{}) {
	getDefaultLogger().log(DEBUG, "", a...)
}
