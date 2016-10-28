// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recovery // import "gopherpit.com/gopherpit/pkg/recovery"

import (
	"fmt"
	"log"
	"runtime/debug"
)

var defualtLogFunc = log.Print

// Notifier defines interface to inject into Service for panic notifications.
type Notifier interface {
	Notify(title, body string) error
}

// Service provides unified way of logging and notifying panic events.
type Service struct {
	Version   string
	BuildInfo string
	LogFunc   func(...interface{})

	Notifier Notifier
}

// Recover is a function that recovers from panic, logs and notifies event.
// It should be used as an argument to defer statement:
//     recoveryService := &recovery.Service{...}
//     go func() {
//         defer recoveryService.Recover()
//  	   ...
//     }
func (s Service) Recover() {
	if err := recover(); err != nil {
		debugInfo := fmt.Sprintf(
			"%s\r\n\r\nversion: %s, build info: %s\r\n\r\n%s",
			err,
			s.Version,
			s.BuildInfo,
			debug.Stack(),
		)
		logFunc := s.LogFunc
		if logFunc == nil {
			logFunc = defualtLogFunc
		}

		logFunc(debugInfo)
		if s.Notifier != nil {
			if err := s.Notifier.Notify(fmt.Sprint("Panic: ", err), debugInfo); err != nil {
				logFunc("recover email sending: ", err)
			}
		}
	}
}
