package logging

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sync"
)

// RotatingFileHandler writes all log messages to file with capability to
// rotate files once they reach defined size.
// If file on provided path does not exist it will be created.
type RotatingFileHandler struct {
	NullHandler

	Level           Level
	Formatter       Formatter
	Directory       string
	FileName        string
	FileExtension   string
	NumberSeparator string
	FileMode        os.FileMode
	MaxFileSize     int64
	MaxFiles        int

	filePath string
	file     *os.File
	writer   *bufio.Writer
	lock     sync.RWMutex
}

// GetLevel returns minimal log level that this handler will process.
func (handler *RotatingFileHandler) GetLevel() Level {
	return handler.Level
}

func (handler *RotatingFileHandler) getFilePath() string {
	if handler.filePath != "" {
		return handler.filePath
	}
	if handler.FileName == "" {
		handler.FileName = "log"
	}
	filename := handler.FileName
	if handler.FileExtension != "" {
		filename = handler.FileName + "." + handler.FileExtension
	}
	handler.filePath = path.Join(handler.Directory, filename)
	return handler.filePath
}

func (handler *RotatingFileHandler) open() error {
	if handler.writer != nil {
		return nil
	}

	file, err := os.OpenFile(handler.getFilePath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, handler.FileMode)

	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		file, err = os.Create(handler.getFilePath())
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
func (handler *RotatingFileHandler) Close() (err error) {
	handler.lock.Lock()
	err = handler.close()
	handler.lock.Unlock()
	return
}

func (handler *RotatingFileHandler) close() error {
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

func (handler *RotatingFileHandler) needsRotation(msg string) bool {
	info, err := os.Stat(handler.getFilePath())
	if err != nil {
		if handler.MaxFiles <= 1 {
			if err != nil {
				return os.IsNotExist(err)
			}
			return false
		} else {
			return true
		}
	}

	if handler.MaxFileSize < 1024 {
		handler.MaxFileSize = 1024
	}

	if info.Size()+int64(len(msg)) >= handler.MaxFileSize {
		return true
	}

	return false
}

func (handler *RotatingFileHandler) rotate() error {
	if err := handler.close(); err != nil {
		return err
	}

	for i := handler.MaxFiles - 2; i >= 0; i-- {
		var fileName string

		if i == 0 {
			fileName = handler.getFilePath()
		} else {
			fileName = path.Join(handler.Directory, fmt.Sprintf("%v%v%v", handler.FileName, handler.NumberSeparator, i))
			if handler.FileExtension != "" {
				fileName = fileName + "." + handler.FileExtension
			}
		}

		_, err := os.Stat(fileName)

		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return err
			}
		}

		nextFileName := path.Join(handler.Directory, fmt.Sprintf("%v%v%v", handler.FileName, handler.NumberSeparator, i+1))
		if handler.FileExtension != "" {
			nextFileName = nextFileName + "." + handler.FileExtension
		}
		_, err = os.Stat(nextFileName)

		if err != nil && !os.IsNotExist(err) {
			err = os.Remove(nextFileName)

			if err != nil {
				return err
			}
		}

		err = os.Rename(fileName, nextFileName)

		if err != nil {
			return err
		}
	}

	return nil
}

// Handle writes message from log record into file.
func (handler *RotatingFileHandler) Handle(record *Record) error {
	handler.lock.Lock()
	defer handler.lock.Unlock()

	msg := handler.Formatter.Format(record) + "\n"

	if handler.needsRotation(msg) {
		if err := handler.rotate(); err != nil {
			return err
		}

		if err := handler.open(); err != nil {
			return err
		}
	}

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
