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

// Service implements gopherpit.com/gopherpit/services/notification.Service
// interface.
type Service struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

// SendEmailResponse is expected structure of JSON-encoded response
// body for SendEmail HTTP request.
type SendEmailResponse struct {
	ID string `json:"id"`
}

// SendEmail sends an e-mail message and returns it's ID. Expected response
// body is a JSON-encoded instance of SendEmailResponse.
func (s Service) SendEmail(email notification.Email) (id string, err error) {
	body, err := json.Marshal(email)
	if err != nil {
		return
	}
	response := &SendEmailResponse{}
	err = s.Client.JSON("POST", "/email", nil, bytes.NewReader(body), response)
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
func (s Service) IsEmailOptedOut(email string) (yes bool, err error) {
	response := &IsEmailOptedOutResponse{}
	err = s.Client.JSON("GET", "/email/opt-out/"+email, nil, nil, response)
	yes = response.Yes
	return
}

// OptOutEmail marks an e-mail address not to send any e-mail messages to.
func (s Service) OptOutEmail(email string) error {
	return s.Client.JSON("POST", "/email/opt-out/"+email, nil, nil, nil)
}

// RemoveOptedOutEmail removes an opt-out mark previosulu set by OptOutEmail.
func (s Service) RemoveOptedOutEmail(email string) error {
	return s.Client.JSON("DELETE", "/email/opt-out/"+email, nil, nil, nil)
}
