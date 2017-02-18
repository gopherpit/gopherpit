// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
}

func (record *Record) process(logger *Logger) {
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
