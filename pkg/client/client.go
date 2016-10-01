// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client provides a generic Client that wraps net/http.Client and
// follows standards within project's REST APIs.
//
// It provides a way to configure exponential retries in Dialer with jitter
// easily configurable usual options and helper functions to be used in
// clients for HTTP APIs.
package client // import "gopherpit.com/gopherpit/pkg/client"

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"resenje.org/marshal"
)

var (
	// DefaultHTTPClient is an instance of net/http.Client that has
	// retry enabled and is used if Client.HTTPClient is nil.
	DefaultHTTPClient = NewHTTPClient(&HTTPClientOptions{
		RetryTimeMax: 45 * time.Second,
	})
	// DefaultKeyHeader is default HTTP header name to pass API key when
	// making a request.
	DefaultKeyHeader = "X-Key"

	random                    = rand.New(rand.NewSource(time.Now().UnixNano()))
	defaultRetrySleepMaxNano  = (2 * time.Second).Nanoseconds()
	defaultRetrySleepBaseNano = (200 * time.Millisecond).Nanoseconds()
)

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
	ErrorRegistry map[int]error
	// HTTPClient is net/http.Client to be used for making HTTP requests.
	// If HTTPClient is nil, DefaultHTTPClient is used.
	HTTPClient *http.Client
}

// HTTPClientOptions is structure that passes optional variables to
// NewHTTPClient.
type HTTPClientOptions struct {
	// Value for net.Dialer.Timeout.
	Timeout time.Duration
	// Value for net.Dialer.KeepAlive.
	KeepAlive time.Duration `envconfig:"KEEP_ALIVE"`
	// Value for net/http.Transport.TLSHandshakeTimeout.
	TLSHandshakeTimeout time.Duration `envconfig:"TLS_HANDSHAKE_TIMEOUT"`
	// Value for crypto/tls.Config.TLSSkipVerify.
	TLSSkipVerify bool `envconfig:"TLS_SKIP_VERIFY"`
	// Maximum time while Dialer reties are made.
	// Default is 0. Which means that Retrying is disabled by default.
	RetryTimeMax time.Duration `envconfig:"RETRY_TIME_MAX"`
	// Maximum time between two retries.
	// Default is 2 seconds.
	RetrySleepMax time.Duration `envconfig:"RETRY_SLEEP_MAX"`
	// Time for first retry. Every other is doubled until RetrySleepMax.
	// Default is 200 milliseconds.
	RetrySleepBase time.Duration `envconfig:"RETRY_SLEEP_BASE"`
}

type httpClientOptionsJSON struct {
	Timeout             marshal.Duration `json:"timeout,omitempty"`
	KeepAlive           marshal.Duration `json:"keep-alive,omitempty"`
	TLSHandshakeTimeout marshal.Duration `json:"tls-handshake-timeout,omitempty"`
	TLSSkipVerify       bool             `json:"tls-skip-verify,omitempty"`
	RetryTimeMax        marshal.Duration `json:"retry-time-max,omitempty"`
	RetrySleepMax       marshal.Duration `json:"retry-sleep-max,omitempty"`
	RetrySleepBase      marshal.Duration `json:"retry-sleep-base,omitempty"`
}

// MarshalJSON implements of json.Marshaler interface.
// It marshals string representations of time.Duration.
func (o HTTPClientOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(httpClientOptionsJSON{
		Timeout:             marshal.Duration(o.Timeout),
		KeepAlive:           marshal.Duration(o.KeepAlive),
		TLSHandshakeTimeout: marshal.Duration(o.TLSHandshakeTimeout),
		TLSSkipVerify:       o.TLSSkipVerify,
		RetryTimeMax:        marshal.Duration(o.RetryTimeMax),
		RetrySleepMax:       marshal.Duration(o.RetrySleepMax),
		RetrySleepBase:      marshal.Duration(o.RetrySleepBase),
	})
}

// UnmarshalJSON implements json.Unamrshaler interface.
// It parses time.Duration as strings.
func (o *HTTPClientOptions) UnmarshalJSON(data []byte) error {
	v := &httpClientOptionsJSON{}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	o = &HTTPClientOptions{
		Timeout:             v.Timeout.Duration(),
		KeepAlive:           v.KeepAlive.Duration(),
		TLSHandshakeTimeout: v.TLSHandshakeTimeout.Duration(),
		TLSSkipVerify:       v.TLSSkipVerify,
		RetryTimeMax:        v.RetryTimeMax.Duration(),
		RetrySleepMax:       v.RetrySleepMax.Duration(),
		RetrySleepBase:      v.RetrySleepBase.Duration(),
	}
	return nil
}

// NewHTTPClient creates a net/http.Client with options from HTTPClientOptions.
func NewHTTPClient(options *HTTPClientOptions) *http.Client {
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if options.KeepAlive == 0 {
		options.KeepAlive = 30 * time.Second
	}
	if options.TLSHandshakeTimeout == 0 {
		options.TLSHandshakeTimeout = 30 * time.Second
	}

	netDialFunc := (&net.Dialer{
		Timeout:   options.Timeout,
		KeepAlive: options.KeepAlive,
	}).Dial

	dialFunc := netDialFunc

	if options.RetryTimeMax > 0 {
		retrySleepBaseNano := options.RetrySleepBase.Nanoseconds()
		if retrySleepBaseNano == 0 {
			retrySleepBaseNano = defaultRetrySleepBaseNano
		}
		retrySleepMaxNano := options.RetrySleepMax.Nanoseconds()
		if retrySleepMaxNano == 0 {
			retrySleepMaxNano = defaultRetrySleepMaxNano
		}
		dialFunc = func(network, address string) (conn net.Conn, err error) {
			var k int64 = 1
			sleepNano := retrySleepBaseNano
			start := time.Now()
			for time.Since(start.Add(-time.Duration(sleepNano))) < options.RetryTimeMax {
				conn, err = netDialFunc(network, address)
				if err != nil {
					sleepNano = retrySleepBaseNano * k
					if sleepNano <= 0 {
						break
					}
					time.Sleep(time.Duration(random.Int63n(func(x, y int64) int64 {
						if x < y {
							return x
						}
						return y
					}(retrySleepMaxNano, sleepNano))))
					k = 2 * k
					continue
				}
				return
			}
			return
		}
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		Dial:                dialFunc,
		TLSHandshakeTimeout: options.TLSHandshakeTimeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: options.TLSSkipVerify},
	}
	http2.ConfigureTransport(transport)
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			if len(via) == 0 {
				return nil
			}
			for attr, val := range via[0].Header {
				if _, ok := req.Header[attr]; !ok {
					req.Header[attr] = val
				}
			}
			return nil
		},
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
		httpClient = DefaultHTTPClient
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

		message := &struct {
			Message *string `json:"message"`
			Code    *int    `json:"code"`
		}{}
		if resp.ContentLength != 0 && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			if err = json.NewDecoder(resp.Body).Decode(message); err != nil {
				if e, ok := err.(*json.SyntaxError); ok {
					err = fmt.Errorf("json: %s, offset: %d", e, e.Offset)
					return
				}
				return
			}
		}
		if message.Code != nil {
			var ok bool
			if err, ok = c.ErrorRegistry[*message.Code]; ok {
				return
			}
		}
		if message.Message != nil {
			err = &Error{
				message: *message.Message,
			}
			return
		}
		err = &HTTPError{
			Status: resp.Status,
			Code:   resp.StatusCode,
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
				return fmt.Errorf("json: %s, offset: %d", e, e.Offset)
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
// whole data is read before closing it.
func (c Client) Stream(method, path string, query url.Values, body io.Reader, accept []string) (data io.ReadCloser, contentType string, err error) {
	resp, err := c.Request(method, path, query, body, accept)
	if err != nil {
		return
	}

	contentType = resp.Header.Get("Content-Type")
	data = resp.Body
	return
}

// Error is a generic error in this package.
type Error struct {
	message string
}

func (e *Error) Error() string {
	return e.message
}

// HTTPError represents a HTTP error that contains status text and status code.
type HTTPError struct {
	// HTTP response status text.
	Status string
	// HTTP response status code.
	Code int
}

func (e *HTTPError) Error() string {
	return e.Status
}
