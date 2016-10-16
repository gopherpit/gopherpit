// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"io"
	"os"
	"sync"
)

// Handler is interface that declares what every logging handler
// must implement in order to be used with this library.
type Handler interface {
	// Handle should accept log record instance and process it.
	Handle(record *Record) error
	// HandleError should process errors that occurred during calls to Handle method
	HandleError(error) error
	// GetLevel should return level that this handler is processing.
	GetLevel() Level
	// Close should free handler resources (opened files, etc)
	Close() error
}

// NullHandler ignores all messages provided to it.
type NullHandler struct{}

// Handle ignores all messages.
func (handler *NullHandler) Handle(record *Record) error {
	return nil
}

// HandleError prints provided error to stderr.
func (handler *NullHandler) HandleError(err error) error {
	os.Stderr.WriteString(err.Error())
	return nil
}

// GetLevel returns current level for this handler.
func (handler *NullHandler) GetLevel() Level {
	return DEBUG
}

// Close does nothing for this handler.
func (handler *NullHandler) Close() error {
	return nil
}

// WriteHandler requires io.Writer and sends all messages to this writer.
type WriteHandler struct {
	NullHandler

	Level     Level
	Formatter Formatter
	Writer    io.Writer
	lock      sync.RWMutex
}

// Handle writes all provided log records to writer provided during creation.
func (handler *WriteHandler) Handle(record *Record) error {
	handler.lock.Lock()
	_, err := handler.Writer.Write([]byte(handler.Formatter.Format(record) + "\n"))
	handler.lock.Unlock()
	return err
}

// GetLevel returns current level for this handler.
func (handler *WriteHandler) GetLevel() Level {
	return handler.Level
}

// MemoryHandler stores all messages in memory.
// If needed, messages can be obtained from Messages field.
type MemoryHandler struct {
	NullHandler

	Level     Level
	Formatter Formatter
	Messages  []string
	lock      sync.RWMutex
}

// Handle appends message to Messages array.
func (handler *MemoryHandler) Handle(record *Record) error {
	handler.lock.Lock()
	handler.Messages = append(handler.Messages, handler.Formatter.Format(record))
	handler.lock.Unlock()
	return nil
}

// GetLevel returns current level for this handler.
func (handler *MemoryHandler) GetLevel() Level {
	return handler.Level
}
