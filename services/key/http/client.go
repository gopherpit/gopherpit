// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpKey provides a HTTP client to an external
// key service that can respond to HTTP requests defined here.
package httpKey // import "gopherpit.com/gopherpit/services/key/http"

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"

	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/key"
)

// Client implements gopherpit.com/gopherpit/services/key.Service interface.
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

// KeyByRef retrieves a Key instance by making a HTTP GET request
// to {Client.Endpoint}/keys/{ref}.
func (c Client) KeyByRef(ref string) (k *key.Key, err error) {
	k = &key.Key{}
	err = c.JSON("GET", "/keys/"+ref, nil, nil, k)
	err = getServiceError(err)
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
func (c Client) KeyBySecret(secret string) (k *key.Key, err error) {
	body, err := json.Marshal(KeyBySecretRequest{
		Secret: secret,
	})
	if err != nil {
		return
	}
	k = &key.Key{}
	err = c.JSON("POST", "/secrets", nil, bytes.NewReader(body), k)
	err = getServiceError(err)
	return
}

// CreateKey creates a new Key with Options by making a HTTP POST
// request to {Client.Endpoint}/keys/{ref}. Post body is a JSON-encoded
// key.Options instance.
func (c Client) CreateKey(ref string, o *key.Options) (k *key.Key, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	k = &key.Key{}
	err = c.JSON("POST", "/keys/"+ref, nil, bytes.NewReader(body), k)
	err = getServiceError(err)
	return
}

// UpdateKey changes the data of an existing Key by making a HTTP PUT
// request to {Client.Endpoint}/keys/{ref}. Post body is a JSON-encoded
// key.Options instance.
func (c Client) UpdateKey(ref string, o *key.Options) (k *key.Key, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	k = &key.Key{}
	err = c.JSON("PUT", "/keys/"+ref, nil, bytes.NewReader(body), k)
	err = getServiceError(err)
	return
}

// DeleteKey deletes an existing Key by making a HTTP DELETE request
// to {Client.Endpoint}/keys/{ref}.
func (c Client) DeleteKey(ref string) error {
	return c.JSON("DELETE", "/keys/"+ref, nil, nil, nil)
}

// RegenerateSecretResponse is a structure that is returned as JSON-encoded body
// from RegenerateSecret HTTP request.
type RegenerateSecretResponse struct {
	Secret string `json:"secret"`
}

// RegenerateSecret generates a new secret key by making a HTTP POST request
// to {Client.Endpoint}/keys/{ref}/secret.
func (c Client) RegenerateSecret(ref string) (secret string, err error) {
	response := &RegenerateSecretResponse{}
	err = c.JSON("POST", "/keys/"+ref+"/secret", nil, nil, response)
	err = getServiceError(err)
	if err != nil {
		return
	}
	secret = response.Secret
	return
}

// Keys retrieves a paginated list of Key instances by making a HTTP GET request
// to {Client.Endpoint}/keys.
func (c Client) Keys(startID string, limit int) (page key.KeysPage, err error) {
	query := url.Values{}
	if startID != "" {
		query.Set("start", startID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/keys", query, nil, page)
	err = getServiceError(err)
	return
}
