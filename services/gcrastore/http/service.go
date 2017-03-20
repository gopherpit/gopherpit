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

type Service struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	return &Service{Client: c}
}

type GetWithTimeResponse struct {
	Value     int64     `json:"value"`
	StoreTime time.Time `json:"store-time"`
}

func (s Service) GetWithTime(key string) (value int64, storeTime time.Time, err error) {
	response := &GetWithTimeResponse{}
	err = s.Client.JSON("GET", "/keys/"+key, nil, nil, response)
	value = response.Value
	storeTime = response.StoreTime
	return
}

type SetResponse struct {
	IsSet bool `json:"is-set"`
}

type SetIfNotExistsWithTTLRequest struct {
	Value int64         `json:"value"`
	TTL   time.Duration `json:"ttl"`
}

func (s Service) SetIfNotExistsWithTTL(key string, value int64, ttl time.Duration) (isSet bool, err error) {
	body, err := json.Marshal(SetIfNotExistsWithTTLRequest{
		Value: value,
		TTL:   ttl,
	})
	if err != nil {
		return
	}
	response := &SetResponse{}
	err = s.Client.JSON("POST", "/keys/"+key, nil, bytes.NewReader(body), response)
	isSet = response.IsSet
	return
}

type CompareAndSwapWithTTLRequest struct {
	Old int64         `json:"old"`
	New int64         `json:"new"`
	TTL time.Duration `json:"ttl"`
}

func (s Service) CompareAndSwapWithTTL(key string, old, new int64, ttl time.Duration) (isSet bool, err error) {
	body, err := json.Marshal(CompareAndSwapWithTTLRequest{
		Old: old,
		New: new,
		TTL: ttl,
	})
	if err != nil {
		return
	}
	response := &SetResponse{}
	err = s.Client.JSON("PUT", "/keys/"+key, nil, bytes.NewReader(body), response)
	isSet = response.IsSet
	return
}
