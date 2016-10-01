package logging

import (
	"bytes"
	"unicode"
)

type infoLogWriter struct {
	logger *Logger
}

func NewInfoLogWriter(logger *Logger) infoLogWriter {
	return infoLogWriter{
		logger: logger,
	}
}

func (lw infoLogWriter) Write(p []byte) (int, error) {
	lw.logger.Info(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return 0, nil
}

type errorLogWriter struct {
	logger *Logger
}

func NewErrorLogWriter(logger *Logger) errorLogWriter {
	return errorLogWriter{
		logger: logger,
	}
}

func (lw errorLogWriter) Write(p []byte) (int, error) {
	lw.logger.Error(string(bytes.TrimRightFunc(p, unicode.IsSpace)))
	return 0, nil
}
