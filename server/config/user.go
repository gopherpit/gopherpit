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

	"github.com/kelseyhightower/envconfig"
)

// UserOptions defines parameters related to the user management.
type UserOptions struct {
	RememberMeDays        int `json:"remember-me-days" envconfig:"REMEMBER_ME_DAYS"`
	PasswordNoReuseMonths int `json:"password-no-reuse-months" envconfig:"PASSWORD_NO_REUSE_MONTHS"`
}

// NewUserOptions initializes UserOptions with default values.
func NewUserOptions() *UserOptions {
	return &UserOptions{
		RememberMeDays:        45,
		PasswordNoReuseMonths: 0,
	}
}

// Update updates options by loading user.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *UserOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "user.json")
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
	data, _ := json.MarshalIndent(o, "", "    ")
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
