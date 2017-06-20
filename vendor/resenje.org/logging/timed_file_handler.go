// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"os"
	"path/filepath"
	"sync"
)

// TimedFileHandler writes all log messages to file with name
// constructed from record time.
// If file or directoeies on provided path do not exist they will be created.
type TimedFileHandler struct {
	NullHandler

	Level          Level
	Formatter      Formatter
	Directory      string
	FileExtension  string
	FilenameLayout string
	FileMode       os.FileMode
	DirectoryMode  os.FileMode

	timestamp string
	file      *os.File
	lock      sync.RWMutex
}

// GetLevel returns minimal log level that this handler will process.
func (handler *TimedFileHandler) GetLevel() Level {
	return handler.Level
}

// Close releases resources used by this handler (file that log messages
// were written into).
func (handler *TimedFileHandler) Close() (err error) {
	handler.lock.Lock()
	err = handler.close()
	handler.lock.Unlock()
	return
}

func (handler *TimedFileHandler) close() error {
	if handler.file != nil {
		if err := handler.file.Close(); err != nil {
			return err
		}
		handler.file = nil
	}
	handler.timestamp = ""
	return nil
}

// Handle writes message from log record into file.
func (handler *TimedFileHandler) Handle(record *Record) (err error) {
	handler.lock.Lock()
	defer handler.lock.Unlock()

	if handler.FilenameLayout == "" {
		handler.FilenameLayout = "2006-01-02"
	}
	timestamp := record.Time.Format(handler.FilenameLayout)
	if handler.timestamp != timestamp || handler.file == nil {
		filename := filepath.Join(handler.Directory, timestamp)
		if handler.FileExtension != "" {

			filename += handler.FileExtension
		}
		if handler.DirectoryMode == 0 {
			handler.DirectoryMode = 0750
		}
		if err = os.MkdirAll(filepath.Dir(filename), handler.DirectoryMode); err != nil {
			return err
		}
		if handler.FileMode == 0 {
			handler.FileMode = 0640
		}
		handler.file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, handler.FileMode)
		if err != nil {
			return err
		}
	}

	msg := handler.Formatter.Format(record) + "\n"

	_, err = handler.file.Write([]byte(msg))
	return
}
