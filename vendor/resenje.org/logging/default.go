package logging

import (
	"os"
)

func init() {
	InitDefaultLogger()
}

// InitDefaultLogger creates new logger that writes log messages to
// standard output in DEBUG level. This logger is automatically created and
// this function needs to be called only if all loggers were removed
// and default logger needs recreating.
func InitDefaultLogger() {
	var err error
	_, err = NewLogger("default", DEBUG, []Handler{
		&WriteHandler{
			Level:     DEBUG,
			Formatter: &StandardFormatter{TimeFormat: StandardTimeFormat},
			Writer:    os.Stderr,
		},
	}, 0)
	if err != nil {
		panic(err)
	}
}

func getDefaultLogger() (l *Logger) {
	lock.RLock()
	l = loggers["default"]
	lock.RUnlock()
	return
}

// Pause temporarily stops processing of log messages in default logger.
func Pause() {
	if logger := getDefaultLogger(); logger != nil {
		logger.Pause()
	}
}

// Unpause continues processing of log messages in default logger.
func Unpause() {
	if logger := getDefaultLogger(); logger != nil {
		logger.Unpause()
	}
}

// Stop permanently stops processing of log messages in default logger.
func Stop() {
	if logger := getDefaultLogger(); logger != nil {
		logger.Stop()
	}
}

// SetLevel sets maximum level for log messages that default logger
// will process.
func SetLevel(level Level) {
	if logger := getDefaultLogger(); logger != nil {
		logger.SetLevel(level)
	}
}

func SetBufferLength(length int) {
	if logger := getDefaultLogger(); logger != nil {
		logger.SetBufferLength(length)
	}
}

// AddHandler adds new handler to default logger.
func AddHandler(handler Handler) {
	if logger := getDefaultLogger(); logger != nil {
		logger.AddHandler(handler)
	}
}

// ClearHandlers remove all handlers from default logger.
func ClearHandlers() {
	if logger := getDefaultLogger(); logger != nil {
		logger.ClearHandlers()
	}
}

// Logf logs provided message with formatting with default logger.
func Logf(level Level, format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(level, format, a...)
	}
}

// Log logs provided message with default logger.
func Log(level Level, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(level, "", a...)
	}
}

// Emergencyf logs provided message with formatting in EMERGENCY level
// with default logger.
func Emergencyf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(EMERGENCY, format, a...)
	}
}

// Emergency logs provided message in EMERGENCY level with default logger.
func Emergency(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(EMERGENCY, "", a...)
	}
}

// Alertf logs provided message with formatting in ALERT level
// with default logger.
func Alertf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(ALERT, format, a...)
	}
}

// Alert logs provided message in ALERT level with default logger.
func Alert(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(ALERT, "", a...)
	}
}

// Criticalf logs provided message with formatting in CRITICAL level
// with default logger.
func Criticalf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(CRITICAL, format, a...)
	}
}

// Critical logs provided message in CRITICAL level with default logger.
func Critical(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(CRITICAL, "", a...)
	}
}

// Errorf logs provided message with formatting in ERROR level
// with default logger.
func Errorf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(ERROR, format, a...)
	}
}

// Error logs provided message in ERROR level with default logger.
func Error(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(ERROR, "", a...)
	}
}

// Warningf logs provided message with formatting in WARNING level
// with default logger.
func Warningf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(WARNING, format, a...)
	}
}

// Warning logs provided message in WARNING level with default logger.
func Warning(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(WARNING, "", a...)
	}
}

// Noticef logs provided message with formatting in NOTICE level
// with default logger.
func Noticef(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(NOTICE, format, a...)
	}
}

// Notice logs provided message in NOTICE level with default logger.
func Notice(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(NOTICE, "", a...)
	}
}

// Infof logs provided message with formatting in INFO level
// with default logger.
func Infof(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(INFO, format, a...)
	}
}

// Info logs provided message in INFO level with default logger.
func Info(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(INFO, "", a...)
	}
}

// Debugf logs provided message with formatting in DEBUG level
// with default logger.
func Debugf(format string, a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(DEBUG, format, a...)
	}
}

// Debug logs provided message in DEBUG level with default logger.
func Debug(a ...interface{}) {
	if logger := getDefaultLogger(); logger != nil {
		logger.log(DEBUG, "", a...)
	}
}
