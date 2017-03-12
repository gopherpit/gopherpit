// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	apiClient "resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/api"
)

// Service is GopherPit HTTP API client service.
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
	c.ErrorRegistry = api.ErrorRegistry
	return &Service{Client: c}
}
