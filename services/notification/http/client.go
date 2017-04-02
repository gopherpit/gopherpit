// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpNotification // import "gopherpit.com/gopherpit/services/notification/http"

import (
	"bytes"
	"encoding/json"

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/notification"
)

// Client implements gopherpit.com/gopherpit/services/notification.Service
// interface.
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

// SendEmailResponse is expected structure of JSON-encoded response
// body for SendEmail HTTP request.
type SendEmailResponse struct {
	ID string `json:"id"`
}

// SendEmail sends an e-mail message and returns it's ID. Expected response
// body is a JSON-encoded instance of SendEmailResponse.
func (c Client) SendEmail(email notification.Email) (id string, err error) {
	body, err := json.Marshal(email)
	if err != nil {
		return
	}
	response := &SendEmailResponse{}
	err = c.JSON("POST", "/email", nil, bytes.NewReader(body), response)
	err = getServiceError(err)
	id = response.ID
	return
}

// IsEmailOptedOutResponse is expected structure of JSON-encoded response
// body for IsEmailOptedOut HTTP request.
type IsEmailOptedOutResponse struct {
	Yes bool `json:"yes"`
}

// IsEmailOptedOut returns true or false if e-mail address is marked not to
// send any e-mail messages to. Expected response body is a JSON-encoded
// instance of IsEmailOptedOutResponse.
func (c Client) IsEmailOptedOut(email string) (yes bool, err error) {
	response := &IsEmailOptedOutResponse{}
	err = c.JSON("GET", "/email/opt-out/"+email, nil, nil, response)
	err = getServiceError(err)
	yes = response.Yes
	return
}

// OptOutEmail marks an e-mail address not to send any e-mail messages to.
func (c Client) OptOutEmail(email string) error {
	return c.JSON("POST", "/email/opt-out/"+email, nil, nil, nil)
}

// RemoveOptedOutEmail removes an opt-out mark previosulu set by OptOutEmail.
func (c Client) RemoveOptedOutEmail(email string) error {
	return c.JSON("DELETE", "/email/opt-out/"+email, nil, nil, nil)
}
