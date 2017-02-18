// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpClient // import "resenje.org/httputils/client/http"

import (
	"crypto/tls"
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"resenje.org/marshal"
)

var (
	// Default is an instance of net/http.Client that has
	// retry enabled and is used if Client.Client is nil.
	Default = New(&Options{
		RetryTimeMax: 45 * time.Second,
	})

	random                    = rand.New(rand.NewSource(time.Now().UnixNano()))
	defaultRetrySleepMaxNano  = (2 * time.Second).Nanoseconds()
	defaultRetrySleepBaseNano = (200 * time.Millisecond).Nanoseconds()
)

// Options is structure that passes optional variables to New function.
type Options struct {
	// Value for net.Dialer.Timeout.
	Timeout time.Duration `envconfig:"TIMEOUT"`
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

// New creates a net/http.Client with options from Options.
func New(options *Options) *http.Client {
	return &http.Client{
		Transport: Transport(options),
	}
}

// Transport creates a net/http.Transport with options from Options.
func Transport(options *Options) *http.Transport {
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
	return transport
}

// optionsJSON is a helper structure to marshal
// duration values into human friendly format.
type optionsJSON struct {
	Timeout             marshal.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	KeepAlive           marshal.Duration `json:"keep-alive,omitempty" yaml:"keep-alive,omitempty"`
	TLSHandshakeTimeout marshal.Duration `json:"tls-handshake-timeout,omitempty" yaml:"tls-handshake-timeout,omitempty"`
	TLSSkipVerify       bool             `json:"tls-skip-verify,omitempty" yaml:"tls-skip-verify,omitempty"`
	RetryTimeMax        marshal.Duration `json:"retry-time-max,omitempty" yaml:"retry-time-max,omitempty"`
	RetrySleepMax       marshal.Duration `json:"retry-sleep-max,omitempty" yaml:"retry-sleep-max,omitempty"`
	RetrySleepBase      marshal.Duration `json:"retry-sleep-base,omitempty" yaml:"retry-sleep-base,omitempty"`
}

// MarshalJSON implements of json.Marshaler interface.
// It marshals string representations of time.Duration.
func (o Options) MarshalJSON() ([]byte, error) {
	return json.Marshal(optionsJSON{
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
func (o *Options) UnmarshalJSON(data []byte) error {
	v := &optionsJSON{}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	*o = Options{
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

// MarshalYAML implements of yaml.Marshaler interface.
// It marshals string representations of time.Duration.
func (o Options) MarshalYAML() (interface{}, error) {
	return optionsJSON{
		Timeout:             marshal.Duration(o.Timeout),
		KeepAlive:           marshal.Duration(o.KeepAlive),
		TLSHandshakeTimeout: marshal.Duration(o.TLSHandshakeTimeout),
		TLSSkipVerify:       o.TLSSkipVerify,
		RetryTimeMax:        marshal.Duration(o.RetryTimeMax),
		RetrySleepMax:       marshal.Duration(o.RetrySleepMax),
		RetrySleepBase:      marshal.Duration(o.RetrySleepBase),
	}, nil
}

// UnmarshalYAML implements yaml.Unamrshaler interface.
// It parses time.Duration as strings.
func (o *Options) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := &optionsJSON{}
	if err := unmarshal(v); err != nil {
		return err
	}
	*o = Options{
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
