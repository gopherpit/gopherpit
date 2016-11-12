// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient // import "resenje.org/httputils/client/api"

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// DefaultKeyHeader is default HTTP header name to pass API key when
// making a request.
var DefaultKeyHeader = "X-Key"

// Client stores properties that defines communication with a HTTP API service.
type Client struct {
	// Endpoint is an URL of the service. (required)
	Endpoint string
	// Key is a single string that is used in request authorization.
	Key string
	// KeyHeader is HTTP header name used to pass Client.Key value.
	// If it is left blank, DefaultKeyHeader is used.
	KeyHeader string
	// UserAgent is a string that will be passed as a value to User-Agent
	// HTTP header.
	UserAgent string
	// Headers is optional additional headers that will be passed on
	// each request.
	Headers map[string]string
	// ErrorRegistry maps error codes to actual errors. It is used to
	// identify errors from the services and pass them as return values.
	ErrorRegistry ErrorRegistry
	// HTTPClient is net/http.Client to be used for making HTTP requests.
	// If Client is nil, DefaultClient is used.
	HTTPClient *http.Client
}

// New returns a new instance of Client with default values.
func New(endpoint string, errorRegistry ErrorRegistry) *Client {
	return &Client{
		Endpoint:      endpoint,
		ErrorRegistry: errorRegistry,
		KeyHeader:     DefaultKeyHeader,
		HTTPClient:    http.DefaultClient,
	}
}

// Request makes a HTTP request based on Client configuration and
// arguments provided.
func (c Client) Request(method, path string, query url.Values, body io.Reader, accept []string) (resp *http.Response, err error) {
	if !strings.HasPrefix(c.Endpoint, "http://") && !strings.HasPrefix(c.Endpoint, "https://") {
		c.Endpoint = "http://" + c.Endpoint
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil {
		return
	}
	u.Path += path

	u.RawQuery = query.Encode()
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return
	}
	for _, a := range accept {
		req.Header.Add("Accept", a)
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}
	if c.Key != "" {
		keyHeader := c.KeyHeader
		if keyHeader == "" {
			keyHeader = DefaultKeyHeader
		}
		req.Header.Set(keyHeader, c.Key)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err = httpClient.Do(req)
	if err != nil {
		return
	}

	if 200 > resp.StatusCode || resp.StatusCode >= 300 {
		defer func() {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()

		message := struct {
			Message string `json:"message"`
			Code    *int   `json:"code"`
		}{}
		if resp.ContentLength != 0 && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			if err = json.NewDecoder(resp.Body).Decode(&message); err != nil {
				switch e := err.(type) {
				case *json.SyntaxError:
					message.Message = fmt.Sprintf("json %s, offset: %d", e, e.Offset)
				case *json.UnmarshalTypeError:
					// If the type of message is not as expected,
					// continue with http based error reporting.
				default:
					return
				}
			}
		}
		if message.Code != nil && c.ErrorRegistry != nil {
			if err = c.ErrorRegistry.Error(*message.Code); err != nil {
				return
			}
		}
		var msg string
		if message.Message != "" {
			msg = message.Message
		} else {
			msg = "http status: " + strings.ToLower(http.StatusText(resp.StatusCode))
		}
		err = &Error{
			Message: msg,
			Code:    resp.StatusCode,
		}
	}

	return
}

// JSON makes a HTTP request that expects application/json response.
// It decodes response body to a `response` argument.
func (c Client) JSON(method, path string, query url.Values, body io.Reader, response interface{}) (err error) {
	resp, err := c.Request(method, path, query, body, []string{"application/json"})
	if err != nil {
		return
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if response != nil {
		if resp.ContentLength == 0 {
			return errors.New("empty response body")
		}
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("unsupported content type: %s", contentType)
		}
		if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
			switch e := err.(type) {
			case *json.SyntaxError:
				return fmt.Errorf("json %s, offset: %d", e, e.Offset)
			case *json.UnmarshalTypeError:
				return fmt.Errorf("expected json %s value but got %s, offset %d", e.Type, e.Value, e.Offset)
			}
			return
		}
	}

	return
}

// Stream makes a HTTP request and returns request body as io.ReadCloser,
// to be able to read long running responses. Returned io.ReadCloser must be
// closed at the end of read. To reuse HTTP connection, make sure that the
// whole data is read before closing the reader.
func (c Client) Stream(method, path string, query url.Values, body io.Reader, accept []string) (data io.ReadCloser, contentType string, err error) {
	resp, err := c.Request(method, path, query, body, accept)
	if err != nil {
		return
	}

	contentType = resp.Header.Get("Content-Type")
	data = resp.Body
	return
}
