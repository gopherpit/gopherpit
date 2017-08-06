// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

// APIOptions defines parameters related to service's API functionality.
type APIOptions struct {
	TrustedProxyCIDRs []string `json:"trusted-proxy-cidrs" yaml:"trusted-proxy-cidrs" envconfig:"TRUSTED_PROXY_CIDRS"`
	ProxyRealIPHeader string   `json:"proxy-real-ip-header" yaml:"proxy-real-ip-header" envconfig:"PROXY_REAL_IP_HEADER"`
	HourlyRateLimit   int      `json:"hourly-rate-limit" yaml:"hourly-rate-limit" envconfig:"HOURLY_RATE_LIMIT"`
	Disabled          bool     `json:"disabled" yaml:"disable" envconfig:"DISABLE"`
}

// NewAPIOptions initializes APIOptions with default values.
func NewAPIOptions() *APIOptions {
	return &APIOptions{
		TrustedProxyCIDRs: []string{
		// "127.0.0.0/8",
		// "::1/128",
		},
		ProxyRealIPHeader: "X-Real-Ip",
		HourlyRateLimit:   0,
		Disabled:          false,
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *APIOptions) VerifyAndPrepare() error { return nil }
