// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
)

// APIOptions defines parameters related to service's API functionality.
type APIOptions struct {
	TrustedProxyCIDRs []string `json:"trusted-proxy-cidrs" yaml:"trusted-proxy-cidrs" envconfig:"TRUSTED_PROXY_CIDRS"`
	ProxyRealIPHeader string   `json:"proxy-real-ip-header" yaml:"proxy-real-ip-header" envconfig:"PROXY_REAL_IP_HEADER"`
	HourlyRateLimit   int      `json:"hourly-rate-limit" yaml:"hourly-rate-limit" envconfig:"HOURLY_RATE_LIMIT"`
	Disable           bool     `json:"disable" yaml:"disable" envconfig:"DISABLE"`
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
		Disable:           false,
	}
}

// Update updates options by loading api.json and api.yaml files.
func (o *APIOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "api.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "api.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_api", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a YAML representation of the options.
func (o *APIOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *APIOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *APIOptions) Prepare() error {
	return nil
}
