// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recovery

import (
	"errors"
	"strings"
	"testing"
)

var errTestNotify = errors.New("test notify")

type LogRecorder struct {
	A []interface{}
}

func (l *LogRecorder) LogFunc(a ...interface{}) {
	l.A = a
}

type NotifyRecorder struct {
	Title string
	Body  string
}

func (n *NotifyRecorder) Notify(title, body string) error {
	if title == "Panic: raise notify error" {
		return errTestNotify
	}
	n.Title = title
	n.Body = body
	return nil
}

func TestService(t *testing.T) {
	logRecorder := &LogRecorder{}
	notifyRecorder := &NotifyRecorder{}
	version := "1.2.3"
	buildInfo := "f12ab4e-build-info"
	recoveryService := Service{
		Version:   version,
		BuildInfo: buildInfo,
		LogFunc:   logRecorder.LogFunc,
		Notifier:  notifyRecorder,
	}
	panicText := "panicking for testing"

	func() {
		defer recoveryService.Recover()
		panic(panicText)
	}()

	if len(logRecorder.A) != 1 {
		t.Errorf("LogFunc message: expected 1 argument, but got %d", len(logRecorder.A))
	} else {
		if !strings.Contains(logRecorder.A[0].(string), panicText) {
			t.Errorf(`LogFunc message: panic text "%s" not found in "%s"`, panicText, logRecorder.A[0].(string))
		}
		if !strings.Contains(logRecorder.A[0].(string), version) {
			t.Errorf(`LogFunc message: version "%s" not found in "%s"`, version, logRecorder.A[0].(string))
		}
		if !strings.Contains(logRecorder.A[0].(string), buildInfo) {
			t.Errorf(`LogFunc message: build info "%s" not found in "%s"`, buildInfo, logRecorder.A[0].(string))
		}
	}

	if !strings.Contains(notifyRecorder.Title, panicText) {
		t.Errorf(`Notify title: panic text "%s" not found in "%s"`, panicText, notifyRecorder.Title)
	}
	if !strings.Contains(notifyRecorder.Body, panicText) {
		t.Errorf(`Notify body: panic text "%s" not found in "%s"`, panicText, notifyRecorder.Body)
	}
	if !strings.Contains(notifyRecorder.Body, version) {
		t.Errorf(`Notify body: version "%s" not found in "%s"`, version, notifyRecorder.Body)
	}
	if !strings.Contains(notifyRecorder.Body, buildInfo) {
		t.Errorf(`Notify body: build info "%s" not found in "%s"`, buildInfo, notifyRecorder.Body)
	}
}

func TestServiceDefaults(t *testing.T) {
	logRecorder := &LogRecorder{}
	notifyRecorder := &NotifyRecorder{}
	version := "1.2.3"
	buildInfo := "f12ab4e-build-info"
	recoveryService := Service{
		Version:   version,
		BuildInfo: buildInfo,
	}
	panicText := "panicking for testing"

	defualtLogFunc = logRecorder.LogFunc

	func() {
		defer recoveryService.Recover()
		panic(panicText)
	}()

	if len(logRecorder.A) != 1 {
		t.Errorf("LogFunc message: expected 1 argument, but got %d", len(logRecorder.A))
	} else {
		if !strings.Contains(logRecorder.A[0].(string), panicText) {
			t.Errorf(`LogFunc message: panic text "%s" not found in "%s"`, panicText, logRecorder.A[0].(string))
		}
		if !strings.Contains(logRecorder.A[0].(string), version) {
			t.Errorf(`LogFunc message: version "%s" not found in "%s"`, version, logRecorder.A[0].(string))
		}
		if !strings.Contains(logRecorder.A[0].(string), buildInfo) {
			t.Errorf(`LogFunc message: build info "%s" not found in "%s"`, buildInfo, logRecorder.A[0].(string))
		}
	}

	if strings.Contains(notifyRecorder.Title, panicText) {
		t.Errorf(`Notify title: panic text "%s" found in "%s"`, panicText, notifyRecorder.Title)
	}
	if strings.Contains(notifyRecorder.Body, panicText) {
		t.Errorf(`Notify body: panic text "%s" found in "%s"`, panicText, notifyRecorder.Body)
	}
	if strings.Contains(notifyRecorder.Body, version) {
		t.Errorf(`Notify body: version "%s" found in "%s"`, version, notifyRecorder.Body)
	}
	if strings.Contains(notifyRecorder.Body, buildInfo) {
		t.Errorf(`Notify body: build info "%s" found in "%s"`, buildInfo, notifyRecorder.Body)
	}
}

func TestNotifyError(t *testing.T) {
	logRecorder := &LogRecorder{}
	notifyRecorder := &NotifyRecorder{}
	version := "1.2.3"
	buildInfo := "f12ab4e-build-info"
	recoveryService := Service{
		Version:   version,
		BuildInfo: buildInfo,
		LogFunc:   logRecorder.LogFunc,
		Notifier:  notifyRecorder,
	}
	panicText := "raise notify error"

	func() {
		defer recoveryService.Recover()
		panic(panicText)
	}()

	if len(logRecorder.A) != 2 {
		t.Errorf("LogFunc message: expected 1 argument, but got %d", len(logRecorder.A))
	} else {
		ts := "recover email sending: "
		if logRecorder.A[0].(string) != ts {
			t.Errorf(`LogFunc message: got log argument 0 "%s", expected "%s"`, logRecorder.A[0].(string), ts)
		}
		if logRecorder.A[1].(error) != errTestNotify {
			t.Errorf(`LogFunc message: got log argument 1 "%v", expected "%v"`, logRecorder.A[1].(error), errTestNotify)
		}
	}
}
