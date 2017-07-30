// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpSession provides a HTTP client to an external
// session service that can respond to HTTP requests defined here.
package httpSession // import "gopherpit.com/gopherpit/services/session/http"

import (
	"bytes"
	"encoding/json"

	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/session"
)

// Client implements gopherpit.com/gopherpit/services/session.Service interface.
type Client struct {
	*apiClient.Client
}

// NewClient creates a new Client.
func NewClient(c *apiClient.Client) *Client {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = errorRegistry
	return &Client{Client: c}
}

// Session retrieves a Session instance by making a HTTP GET request
// to {Client.Endpoint}/sessions/{id}.
func (c Client) Session(id string) (ses *session.Session, err error) {
	ses = &session.Session{}
	err = c.JSON("GET", "/sessions/"+id, nil, nil, ses)
	err = getServiceError(err)
	return
}

// CreateSession creates a new Session with Options by making a HTTP POST
// request to {Client.Endpoint}/sessions. Post body is a JSON-encoded
// session.Options instance.
func (c Client) CreateSession(o *session.Options) (ses *session.Session, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	ses = &session.Session{}
	err = c.JSON("POST", "/sessions", nil, bytes.NewReader(body), ses)
	err = getServiceError(err)
	return
}

// UpdateSession changes the data of an existing Session by making a HTTP POST
// request to {Client.Endpoint}/sessions/{id}. Post body is a JSON-encoded
// session.Options instance.
func (c Client) UpdateSession(id string, o *session.Options) (ses *session.Session, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	ses = &session.Session{}
	err = c.JSON("POST", "/sessions/"+id, nil, bytes.NewReader(body), ses)
	err = getServiceError(err)
	return
}

// DeleteSession deletes an existing Session by making a HTTP DELETE request
// to {Client.Endpoint}/sessions/{id}.
func (c Client) DeleteSession(id string) error {
	return c.JSON("DELETE", "/sessions/"+id, nil, nil, nil)
}
