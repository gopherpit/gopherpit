// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"resenje.org/web"

	"gopherpit.com/gopherpit/services/user"
)

type auditRecord struct {
	Time     time.Time   `json:"time,omitempty"`
	UserID   string      `json:"user-id,omitempty"`
	Username string      `json:"username,omitempty"`
	Email    string      `json:"email,omitempty"`
	Info     interface{} `json:"info,omitempty"`
	IPs      string      `json:"ips,omitempty"`
	Action   string      `json:"action,omitempty"`
	Message  string      `json:"massage,omitempty"`
}

func (s *Server) audit(r *http.Request, info interface{}, action, message string) {
	if s.AuditLogger == nil {
		return
	}
	var userID, username, email string
	u, _, err := s.getRequestUser(r)
	if err != nil && err != user.ErrUserNotFound {
		s.Logger.Errorf("audit: get user: %s", err)
		return
	}
	if u != nil {
		userID = u.ID
		username = u.Username
		email = u.Email
	}
	record, err := json.Marshal(auditRecord{
		Time:     time.Now().UTC(),
		UserID:   userID,
		Username: username,
		Email:    email,
		Info:     info,
		IPs:      web.GetRequestIPs(r),
		Action:   action,
		Message:  message,
	})
	if err != nil {
		s.Logger.Errorf("audit: json encode: %s", err)
		return
	}
	s.AuditLogger.Info(string(record))
}

func (s *Server) auditf(r *http.Request, info interface{}, action, format string, a ...interface{}) {
	s.audit(r, info, action, fmt.Sprintf(format, a...))
}
