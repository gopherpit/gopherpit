// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpGCRAStore // import "gopherpit.com/gopherpit/services/gcrastore/http"

import (
	"bytes"
	"encoding/json"
	"time"

	"resenje.org/httputils/client/api"
)

// Service is HTTP implementation of gcrastore.Service.
type Service struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

// NewService initializes a new Service with optional API Client.
// If API Client is nil a default apiClient is used.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	return &Service{Client: c}
}

// GetWithTimeResponse holds information that is returned by GetWithTime
// API endpoint.
type GetWithTimeResponse struct {
	Value     int64     `json:"value"`
	StoreTime time.Time `json:"store-time"`
}

// GetWithTime retrieves value and store time by making a HTTP GET request
// to {Client.Endpoint}/keys/{key}.
func (s Service) GetWithTime(key string) (value int64, storeTime time.Time, err error) {
	response := &GetWithTimeResponse{}
	err = s.Client.JSON("GET", "/keys/"+key, nil, nil, response)
	value = response.Value
	storeTime = response.StoreTime
	return
}

// IsSetResponse holds response information from SetIfNotExistsWithTTL and
// CompareAndSwapWithTTL methods.
type IsSetResponse struct {
	IsSet bool `json:"is-set"`
}

// SetIfNotExistsWithTTLRequest holds information sent in SetIfNotExistsWithTTL
// method.
type SetIfNotExistsWithTTLRequest struct {
	Value int64         `json:"value"`
	TTL   time.Duration `json:"ttl"`
}

// SetIfNotExistsWithTTL sends value and TTL by making a HTTP POST request
// to {Client.Endpoint}/keys/{key}.
func (s Service) SetIfNotExistsWithTTL(key string, value int64, ttl time.Duration) (isSet bool, err error) {
	body, err := json.Marshal(SetIfNotExistsWithTTLRequest{
		Value: value,
		TTL:   ttl,
	})
	if err != nil {
		return
	}
	response := &IsSetResponse{}
	err = s.Client.JSON("POST", "/keys/"+key, nil, bytes.NewReader(body), response)
	isSet = response.IsSet
	return
}

// CompareAndSwapWithTTLRequest holds information sent in CompareAndSwapWithTTL
// method.
type CompareAndSwapWithTTLRequest struct {
	Old int64         `json:"old"`
	New int64         `json:"new"`
	TTL time.Duration `json:"ttl"`
}

// CompareAndSwapWithTTL sends old and new value and TTL by making a HTTP PUT request
// to {Client.Endpoint}/keys/{key}.
func (s Service) CompareAndSwapWithTTL(key string, old, new int64, ttl time.Duration) (isSet bool, err error) {
	body, err := json.Marshal(CompareAndSwapWithTTLRequest{
		Old: old,
		New: new,
		TTL: ttl,
	})
	if err != nil {
		return
	}
	response := &IsSetResponse{}
	err = s.Client.JSON("PUT", "/keys/"+key, nil, bytes.NewReader(body), response)
	isSet = response.IsSet
	return
}
