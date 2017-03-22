// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client // import "gopherpit.com/gopherpit/client"

import (
	apiClient "resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/api"
)

// Client is GopherPit HTTP API client service.
type Client struct {
	// Client provides HTTP request making functionality.
	Client *apiClient.Client
}

// NewClient creates a new Client and injects api.ErrorRegistry
// in the API Client.
func NewClient(c *apiClient.Client) *Client {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = api.ErrorRegistry
	return &Client{Client: c}
}
