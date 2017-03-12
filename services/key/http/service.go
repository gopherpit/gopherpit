// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpKey provides a Service that is a HTTP client to an external
// key service that can respond to HTTP requests defined here.
package httpKey // import "gopherpit.com/gopherpit/services/key/http"

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/key"
)

// Service implements gopherpit.com/gopherpit/services/key.Service interface.
type Service struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

// NewService creates a new Service and injects key.ErrorRegistry
// in the API Client.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = key.ErrorRegistry
	return &Service{Client: c}
}

// KeyByRef retrieves a Key instance by making a HTTP GET request
// to {Client.Endpoint}/keys/{ref}.
func (s Service) KeyByRef(ref string) (k *key.Key, err error) {
	k = &key.Key{}
	err = s.Client.JSON("GET", "/keys/"+ref, nil, nil, k)
	return
}

// KeyBySecretRequest is a structure that is passed as JSON-encoded body
// to KeyBySecret HTTP request.
type KeyBySecretRequest struct {
	Secret string `json:"secret"`
}

// KeyBySecret validates a password of an existing User by making a HTTP POST
// request to {Client.Endpoint}/authenticate. Request body is a
// JSON-encoded KeyBySecretRequest instance. Expected response body is a
// JSON-encoded instance of key.Key.
func (s Service) KeyBySecret(secret string) (k *key.Key, err error) {
	body, err := json.Marshal(KeyBySecretRequest{
		Secret: secret,
	})
	if err != nil {
		return
	}
	k = &key.Key{}
	err = s.Client.JSON("POST", "/secrets", nil, bytes.NewReader(body), k)
	return
}

// CreateKey creates a new Key with Options by making a HTTP POST
// request to {Client.Endpoint}/keys/{ref}. Post body is a JSON-encoded
// key.Options instance.
func (s Service) CreateKey(ref string, o *key.Options) (k *key.Key, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	k = &key.Key{}
	err = s.Client.JSON("POST", "/keys/"+ref, nil, bytes.NewReader(body), k)
	return
}

// UpdateKey changes the data of an existing Key by making a HTTP PUT
// request to {Client.Endpoint}/keys/{ref}. Post body is a JSON-encoded
// key.Options instance.
func (s Service) UpdateKey(ref string, o *key.Options) (k *key.Key, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	k = &key.Key{}
	err = s.Client.JSON("PUT", "/keys/"+ref, nil, bytes.NewReader(body), k)
	return
}

// DeleteKey deletes an existing Key by making a HTTP DELETE request
// to {Client.Endpoint}/keys/{ref}.
func (s Service) DeleteKey(ref string) error {
	return s.Client.JSON("DELETE", "/keys/"+ref, nil, nil, nil)
}

// RegenerateSecretResponse is a structure that is returned as JSON-encoded body
// from RegenerateSecret HTTP request.
type RegenerateSecretResponse struct {
	Secret string `json:"secret"`
}

// RegenerateSecret generates a new secret key by making a HTTP POST request
// to {Client.Endpoint}/keys/{ref}/secret.
func (s Service) RegenerateSecret(ref string) (secret string, err error) {
	response := &RegenerateSecretResponse{}
	err = s.Client.JSON("POST", "/keys/"+ref+"/secret", nil, nil, response)
	if err != nil {
		return
	}
	secret = response.Secret
	return
}

// Keys retrieves a paginated list of Key instances by making a HTTP GET request
// to {Client.Endpoint}/keys.
func (s Service) Keys(startID string, limit int) (page key.KeysPage, err error) {
	query := url.Values{}
	if startID != "" {
		query.Set("start", startID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/keys", query, nil, page)
	return
}
