package logging

import (
	"sync/atomic"
	"time"
)

// Record is representation of single message that needs to be logged in single handler.
type Record struct {
	Time    time.Time `json:"time"`
	Level   Level     `json:"level"`
	Message string    `json:"message"`
	logger  *Logger
}

func (record *Record) process() {
	logger := record.logger
	logger.lock.RLock()

	if record.Level <= logger.Level {
		for _, handler := range logger.Handlers {
			if record.Level <= handler.GetLevel() {
				if err := handler.Handle(record); err != nil {
					handler.HandleError(err)
				}
			}
		}
	}

	logger.lock.RUnlock()
	atomic.AddUint64(&logger.countOut, 1)
}
