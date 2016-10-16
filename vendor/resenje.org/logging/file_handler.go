// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"bufio"
	"os"
	"sync"
)

// FileHandler is a handler that writes log messages to file.
// If file on provided path does not exist it will be created.
//
// Note that this handler does not perform any kind of file rotation or
// truncation, so files will grow indefinitely. You might want to check out
// RotatingFileHandler and
type FileHandler struct {
	NullHandler

	Level     Level
	Formatter Formatter
	FilePath  string
	FileMode  os.FileMode

	file   *os.File
	writer *bufio.Writer
	lock   sync.RWMutex
}

// GetLevel returns minimal log level that this handler will process.
func (handler *FileHandler) GetLevel() Level {
	return handler.Level
}

func (handler *FileHandler) open() error {
	if handler.writer != nil {
		return nil
	}

	file, err := os.OpenFile(handler.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, handler.FileMode)

	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		file, err = os.Create(handler.FilePath)
		if err != nil {
			return err
		}
	}

	handler.file = file
	handler.writer = bufio.NewWriter(handler.file)

	return nil
}

// Close releases resources used by this handler (file that log messages
// were written into).
func (handler *FileHandler) Close() (err error) {
	handler.lock.Lock()
	err = handler.close()
	handler.lock.Unlock()
	return
}

func (handler *FileHandler) close() error {
	if handler.writer != nil {
		if err := handler.writer.Flush(); err != nil {
			return err
		}
		handler.writer = nil
	}
	if handler.file != nil {
		if err := handler.file.Close(); err != nil {
			return err
		}
		handler.file = nil
	}
	return nil
}

// Handle writes message from log record into file.
func (handler *FileHandler) Handle(record *Record) error {
	handler.lock.Lock()
	defer handler.lock.Unlock()

	msg := handler.Formatter.Format(record) + "\n"

	if handler.writer == nil {

		if err := handler.open(); err != nil {
			return err
		}
	}

	_, err := handler.writer.Write([]byte(msg))
	if err != nil {
		return err
	}
	handler.writer.Flush()

	return nil
}
