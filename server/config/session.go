// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
	"resenje.org/marshal"
)

// SessionOptions defines parameters related to session storage.
type SessionOptions struct {
	CleanupPeriod   marshal.Duration `json:"cleanup-period" yaml:"cleanup-period" envconfig:"CLEANUP_PERIOD"`
	DefaultLifetime marshal.Duration `json:"default-lifetime" yaml:"default-lifetime" envconfig:"DEFAULT_LIFETIME"`
}

// NewSessionOptions initializes SessionOptions with default values.
func NewSessionOptions() *SessionOptions {
	return &SessionOptions{
		CleanupPeriod:   marshal.Duration(25 * time.Hour),
		DefaultLifetime: marshal.Duration(30 * 24 * time.Hour),
	}
}

// Update updates options by loading session.json files.
func (o *SessionOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "session.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "session.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_session", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a JSON representation of the options.
func (o *SessionOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *SessionOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *SessionOptions) Prepare() error {
	return nil
}
