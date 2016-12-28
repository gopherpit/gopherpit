// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpSession provides a Service that is a HTTP client to an external
// session service that can respond to HTTP requests defined here.
package httpSession // import "gopherpit.com/gopherpit/services/session/http"

import (
	"bytes"
	"encoding/json"

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/session"
)

// Service implements gopherpit.com/gopherpit/services/session.Service interface.
type Service struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

// NewService creates a new Service and injects session.ErrorRegistry
// in the API Client.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = session.ErrorRegistry
	return &Service{Client: c}
}

// Session retrieves a Session instance by making a HTTP GET request
// to {Client.Endpoint}/sessions/{id}.
func (s Service) Session(id string) (ses *session.Session, err error) {
	err = s.Client.JSON("GET", "/sessions/"+id, nil, nil, ses)
	return
}

// CreateSession creates a new Session with Options by making a HTTP POST
// request to {Client.Endpoint}/sessions. Post body is a JSON-encoded
// session.Options instance.
func (s Service) CreateSession(o *session.Options) (ses *session.Session, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/sessions", nil, bytes.NewReader(body), ses)
	return
}

// UpdateSession changes the data of an existing Session by making a HTTP POST
// request to {Client.Endpoint}/sessions/{id}. Post body is a JSON-encoded
// session.Options instance.
func (s Service) UpdateSession(id string, o *session.Options) (ses *session.Session, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/sessions/"+id, nil, bytes.NewReader(body), ses)
	return
}

// DeleteSession deletes an existing Session by making a HTTP DELETE request
// to {Client.Endpoint}/sessions/{id}.
func (s Service) DeleteSession(id string) error {
	return s.Client.JSON("DELETE", "/sessions/"+id, nil, nil, nil)
}
