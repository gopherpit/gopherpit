// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"resenje.org/marshal"
)

// SessionOptions defines parameters related to session storage.
type SessionOptions struct {
	CleanupPeriod   marshal.Duration `json:"cleanup-period" envconfig:"CLEANUP_PERIOD"`
	DefaultLifetime marshal.Duration `json:"default-lifetime" envconfig:"DEFAULT_LIFETIME"`
}

// NewSessionOptions initializes SessionOptions with default values.
func NewSessionOptions() *SessionOptions {
	return &SessionOptions{
		CleanupPeriod:   marshal.Duration(25 * time.Hour),
		DefaultLifetime: marshal.Duration(30 * 24 * time.Hour),
	}
}

// Update updates options by loading session.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *SessionOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "session.json")
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
	data, _ := json.MarshalIndent(o, "", "    ")
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
