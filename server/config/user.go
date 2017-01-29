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

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
)

// UserOptions defines parameters related to the user management.
type UserOptions struct {
	RememberMeDays        int `json:"remember-me-days" yaml:"remember-me-days" envconfig:"REMEMBER_ME_DAYS"`
	PasswordNoReuseMonths int `json:"password-no-reuse-months" yaml:"password-no-reuse-months" envconfig:"PASSWORD_NO_REUSE_MONTHS"`
}

// NewUserOptions initializes UserOptions with default values.
func NewUserOptions() *UserOptions {
	return &UserOptions{
		RememberMeDays:        45,
		PasswordNoReuseMonths: 0,
	}
}

// Update updates options by loading user.json files.
func (o *UserOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "user.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "user.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_user", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a JSON representation of the options.
func (o *UserOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *UserOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *UserOptions) Prepare() error {
	return nil
}
