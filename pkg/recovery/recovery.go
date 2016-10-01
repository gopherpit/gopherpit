// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recovery // import "gopherpit.com/gopherpit/pkg/recovery"

import (
	"fmt"
	"runtime/debug"

	"resenje.org/logging"

	"gopherpit.com/gopherpit/pkg/email"
)

// Service provides unified way of logging and notifying panic events.
type Service struct {
	Version   string
	BuildInfo string

	EmailService *email.Service
}

// Recover is a fucntion that recovers from panic, logs and notifies event.
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
		logging.Error(debugInfo)
		if s.EmailService != nil && len(s.EmailService.NotifyAddresses) > 0 {
			if err := s.EmailService.Notify(fmt.Sprint("Panic: ", err), debugInfo); err != nil {
				logging.Error("recover email sending: ", err)
			}
		}
	}
}
