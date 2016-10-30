// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging // import "resenje.org/logging"

import (
	"container/ring"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	loggers = make(map[string]*Logger)
	lock    = &sync.RWMutex{}
)

// Logger is struct that is capable of processing log messages.
// It is central point of application logging configuration. Application
// can have multiple loggers with different names and at moment of logging
// logger is chosen.
//
// Each logger has level and ignores all log records below defined
// level. Also, each logger has array of handlers used to to actuall log
// record processing.
type Logger struct {
	Name          string
	Level         Level
	Handlers      []Handler
	buffer        *ring.Ring
	stateChannel  chan uint8
	recordChannel chan *Record
	lock          sync.RWMutex
	countIn       uint64
	countOut      uint64
}

// NewLogger creates and returns new logger instance with provided name, log level,
// handlers and buffer length.
// If a logger with the same name already exists, it will be replaced with a new one.
// Log level is lowest level or log record that this logger will handle.
// Log records (above defined log level) will be passed to all log handlers for processing.
func NewLogger(name string, level Level, handlers []Handler, bufferLength int) (l *Logger) {
	l = &Logger{
		Name:          name,
		Level:         level,
		Handlers:      handlers,
		buffer:        ring.New(bufferLength),
		stateChannel:  make(chan uint8, 0),
		recordChannel: make(chan *Record, 2048),
		lock:          sync.RWMutex{},
		countOut:      0,
		countIn:       0,
	}
	go l.run()
	if name == "default" {
		defaultLogger = l
	}
	lock.Lock()
	loggers[name] = l
	lock.Unlock()
	return
}

// GetLogger returns logger instance based on provided name.
// If logger does not exist, error will be returned.
func GetLogger(name string) (*Logger, error) {
	if name == "default" {
		return getDefaultLogger(), nil
	}
	lock.RLock()
	logger, ok := loggers[name]
	lock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown logger %s", name)
	}
	return logger, nil
}

// RemoveLogger deletes logger from global logger registry.
func RemoveLogger(name string) {
	lock.Lock()
	delete(loggers, name)
	if name == "default" {
		defaultLogger = nil
	}
	lock.Unlock()
}

// RemoveLoggers removes all loggers from global logger registry.
// After calling this, not loggers exist, so new ones has to be creaded
// for logging to work.
func RemoveLoggers() {
	lock.Lock()
	loggers = make(map[string]*Logger)
	lock.Unlock()
}

// WaitForAllUnprocessedRecords blocks execution until all unprocessed
// log records are processed.
// Since this library implements async logging, it is possible to have
// unprocessed logs at the moment when application is terminating.
// In that case, log messages can be lost. This mehtods blocks execution
// until all log records are processed, to ensure that all log messages
// are handled.
func WaitForAllUnprocessedRecords() {
	lock.Lock()
	var wg sync.WaitGroup
	for _, logger := range loggers {
		wg.Add(1)
		go func(logger *Logger) {
			logger.WaitForUnprocessedRecords()
			wg.Done()
		}(logger)
	}
	wg.Wait()
	lock.Unlock()
}

// String returns logger name.
func (logger *Logger) String() string {
	return logger.Name
}

// run starts async log records processing.
func (logger *Logger) run() {
	defer func() {
		logger.WaitForUnprocessedRecords()
		logger.closeHandlers()
	}()
recordLoop:
	for {
		select {
		case record := <-logger.recordChannel:
			record.process()
		case state := <-logger.stateChannel:
			switch state {
			case stopped:
				break recordLoop
			case paused:
			stateLoop:
				for {
					select {
					case state := <-logger.stateChannel:
						switch state {
						case stopped:
							break recordLoop
						case running:
							break stateLoop
						default:
							continue
						}
					}
				}
			}
		}
	}
}

// WaitForUnprocessedRecords block execution until all unprocessed log records
// for this logger are processed.
// In order to wait for processing in all loggers, logging.WaitForAllUnprocessedRecords
// can be used.
func (logger *Logger) WaitForUnprocessedRecords() {
	runtime.Gosched()
	var (
		diff     uint64
		diffPrev uint64
		i        uint8
	)
	for {
		diff = atomic.LoadUint64(&logger.countIn) - atomic.LoadUint64(&logger.countOut)
		if diff == diffPrev {
			i++
		}
		if i >= 100 {
			return
		}
		if diff > 0 {
			diffPrev = diff
			time.Sleep(10 * time.Millisecond)
		} else {
			return
		}
	}
}

// closeHandlers calls Close for all handlers for this logger
// to release resources.
func (logger *Logger) closeHandlers() {
	for _, handler := range logger.Handlers {
		handler.Close()
	}
}

// Pause temporarily stops processing of log messages in current logger.
func (logger *Logger) Pause() {
	logger.stateChannel <- paused
}

// Unpause continues processing of log messages in current logger.
func (logger *Logger) Unpause() {
	logger.stateChannel <- running
}

// Stop permanently stops processing of log messages in current logger.
func (logger *Logger) Stop() {
	logger.stateChannel <- stopped
}

// SetBufferLength sets length of buffer for accepting log records.
func (logger *Logger) SetBufferLength(length int) {
	logger.lock.Lock()

	if length == 0 {
		logger.buffer = nil
	} else if length != logger.buffer.Len() {
		logger.buffer = ring.New(length)
	}

	logger.lock.Unlock()
}

// AddHandler add new handler to current logger.
func (logger *Logger) AddHandler(handler Handler) {
	logger.lock.Lock()
	logger.Handlers = append(logger.Handlers, handler)
	logger.flushBuffer()
	logger.lock.Unlock()
}

// ClearHandlers removes all handlers from current logger.
func (logger *Logger) ClearHandlers() {
	logger.lock.Lock()
	logger.closeHandlers()
	logger.Handlers = make([]Handler, 0)
	logger.flushBuffer()
	logger.lock.Unlock()
}

// SetLevel sets lower level that current logger will process.
func (logger *Logger) SetLevel(level Level) {
	logger.lock.Lock()
	logger.Level = level
	logger.flushBuffer()
	logger.lock.Unlock()
}

func (logger *Logger) flushBuffer() {
	if logger.buffer != nil {
		oldBuffer := logger.buffer
		logger.buffer = ring.New(oldBuffer.Len())

		go func() {
			oldBuffer.Do(func(x interface{}) {

				if x == nil {
					return
				}

				record := x.(*Record)

				atomic.AddUint64(&logger.countIn, 1)
				logger.recordChannel <- record
			})
		}()
	}
}

// log creates log record and submits it for processing.
func (logger *Logger) log(level Level, format string, a ...interface{}) {
	var message string
	if format == "" {
		message = fmt.Sprint(a...)
	} else {
		message = fmt.Sprintf(format, a...)
	}

	atomic.AddUint64(&logger.countIn, 1)
	logger.recordChannel <- &Record{
		Level:   level,
		Message: message,
		Time:    time.Now(),
		logger:  logger,
	}
}

// Logf logs provided message with formatting.
func (logger *Logger) Logf(level Level, format string, a ...interface{}) {
	logger.log(level, format, a...)
}

// Log logs provided message.
func (logger *Logger) Log(level Level, a ...interface{}) {
	logger.log(level, "", a...)
}

// Emergencyf logs provided message with formatting in EMERGENCY level.
func (logger *Logger) Emergencyf(format string, a ...interface{}) {
	logger.log(EMERGENCY, format, a...)
}

// Emergency logs provided message in EMERGENCY level.
func (logger *Logger) Emergency(a ...interface{}) {
	logger.log(EMERGENCY, "", a...)
}

// Alertf logs provided message with formatting in ALERT level.
func (logger *Logger) Alertf(format string, a ...interface{}) {
	logger.log(ALERT, format, a...)
}

// Alert logs provided message in ALERT level.
func (logger *Logger) Alert(a ...interface{}) {
	logger.log(ALERT, "", a...)
}

// Criticalf logs provided message with formatting in CRITICAL level.
func (logger *Logger) Criticalf(format string, a ...interface{}) {
	logger.log(CRITICAL, format, a...)
}

// Critical logs provided message in CRITICAL level.
func (logger *Logger) Critical(a ...interface{}) {
	logger.log(CRITICAL, "", a...)
}

// Errorf logs provided message with formatting in ERROR level.
func (logger *Logger) Errorf(format string, a ...interface{}) {
	logger.log(ERROR, format, a...)
}

// Error logs provided message in ERROR level.
func (logger *Logger) Error(a ...interface{}) {
	logger.log(ERROR, "", a...)
}

// Warningf logs provided message with formatting in WARNING level.
func (logger *Logger) Warningf(format string, a ...interface{}) {
	logger.log(WARNING, format, a...)
}

// Warning logs provided message in WARNING level.
func (logger *Logger) Warning(a ...interface{}) {
	logger.log(WARNING, "", a...)
}

// Noticef logs provided message with formatting in NOTICE level.
func (logger *Logger) Noticef(format string, a ...interface{}) {
	logger.log(NOTICE, format, a...)
}

// Notice logs provided message in NOTICE level.
func (logger *Logger) Notice(a ...interface{}) {
	logger.log(NOTICE, "", a...)
}

// Infof logs provided message with formatting in INFO level.
func (logger *Logger) Infof(format string, a ...interface{}) {
	logger.log(INFO, format, a...)
}

// Info logs provided message in INFO level.
func (logger *Logger) Info(a ...interface{}) {
	logger.log(INFO, "", a...)
}

// Debugf logs provided message with formatting in DEBUG level.
func (logger *Logger) Debugf(format string, a ...interface{}) {
	logger.log(DEBUG, format, a...)
}

// Debug logs provided message in DEBUG level.
func (logger *Logger) Debug(a ...interface{}) {
	logger.log(DEBUG, "", a...)
}
