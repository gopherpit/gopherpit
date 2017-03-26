// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package client is a Go client for the GopherPit API.

For more information about the Engine API, see the documentation:
https://gopherpit.com/docs/api

Authorization is performed by Personal Access Token which can
be obtained on Settings page of the GopherPit site.

Usage

Communication with the API is performed by creating a Client
object and calling methods on it.

Example:

    package main

    import (
        "fmt"
        "os"

        "gopherpit.com/gopherpit/client"
    )

    func main() {
        c := client.NewClient(os.Getenv("GOPHERPIT_TOKEN"))

        domains, err := c.Domains("", 0)
        if err != nil {
            fmt.Fprintln(os.Stderr, "get domains", err)
            os.Exit(1)
        }

        for _, domain := range domains.Domains {
            fmt.Printf("%s %s\n", domain.ID, domain.FQDN)
        }
    }

To use GopherPit installation on-premises:

    c := client.NewClientWithEndpoint("https://go.example.com/api/v1", "TOKEN")

*/
package client // import "gopherpit.com/gopherpit/client"

import (
	apiClient "resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/api"
)

var (
	version           = "0.1"
	userAgent         = "GopherPitClient/" + version
	gopherpitEndpoint = "https://gopherpit.com/api/v1"
)

// Client is the API client that performs all operations
// against GopherPit server.
type Client struct {
	apiClient.Client
}

// NewClient creates a new Client object with
// gopherpit.com endpoint. It is intended for connecting to
// publicly available GopherPit service.
func NewClient(key string) *Client {
	return NewClientWithEndpoint(gopherpitEndpoint, key)
}

// NewClientWithEndpoint creates a new Client object with
// HTTP endpoint and Personal Access Token as a key. It is intended
// for connecting to on-premises GopherPit installations.
// Endpoint URL must include schema, host and path components.
// For example: https://go.example.com/api/v1
func NewClientWithEndpoint(endpoint, key string) *Client {
	return newClientWithAPIClient(apiClient.Client{
		Endpoint:  endpoint,
		Key:       key,
		UserAgent: userAgent,
	})
}

// newClientWithAPIClient creates a new Client and injects
// api.ErrorRegistry in the API Client.
func newClientWithAPIClient(c apiClient.Client) *Client {
	c.ErrorRegistry = api.ErrorRegistry
	return &Client{Client: c}
}
