// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config holds project and service related data and structures
// that define optional parameters for different parts of the service.
package config // import "gopherpit.com/gopherpit/server/config"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	// Name is the name of the service.
	Name = "gopherpit"
	// Description is a service description.
	Description = ""

	// Author is the name of the service's author.
	Author = "Janoš Guljaš"
	// AuthorEmail is a contact address of the author.
	AuthorEmail = "janos@resenje.org"

	// Version is a string representing service's version.
	// Set the version on build with: go build -ldflags "-X gopherpit.com/gopherpit/server/config.Version=$(VERSION)"
	Version = "0"
	// BuildInfo is usually a git commit short hash.
	// Set the version on build with: go build -ldflags "-X gopherpit.com/gopherpit/server/config.BuildInfo=$(shell git describe --tags --long --dirty --always)"
	BuildInfo = ""

	// UserAgent is a value for User-Agent HTTP request header value.
	UserAgent = func() string {
		if BuildInfo != "" {
			return fmt.Sprintf("%s/%s-%s", Name, Version, BuildInfo)
		}
		return fmt.Sprintf("%s/%s", Name, Version)
	}()

	// BaseDir is the directory where the service's executable is located.
	BaseDir = func() string {
		baseDir := filepath.Dir(os.Args[0])
		baseDir, err := filepath.Abs(baseDir)
		if err != nil {
			panic(err)
		}
		return baseDir
	}()

	defaultsDir = filepath.Join(BaseDir, "defaults")

	// ConfigDir is default directory where configuration files are located.
	// Set the version on build with: go build -ldflags "-X gopherpit.com/gopherpit/server/config.ConfigDir=$(CONFIG_DIR)"
	ConfigDir = "/etc/" + Name
)

// Options interface defines functionality to update, verify, prepare
// and display configuration.
type Options interface {
	Update(configDir string) error
	Verify() (help string, err error)
	Prepare() error
	String() string
}

// Prepare prepares directories provided in configuration options.
func Prepare(options []Options) error {
	for _, o := range options {
		if err := o.Prepare(); err != nil {
			return err
		}
	}
	return nil
}

// Update updates configuration options from external files.
func Update(options []Options, configDir string) error {
	for _, o := range options {
		if err := o.Update(configDir); err != nil {
			return err
		}
	}
	return nil
}

// Verify verifies configuration values.
func Verify(options []Options) (help string, err error) {
	for _, o := range options {
		if help, err = o.Verify(); err != nil {
			return
		}
	}
	return
}

func loadJSON(filename string, o Options) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}
	if err = json.Unmarshal(data, o); err != nil {
		getLineColFromOffset := func(data []byte, offset int64) (line, col int) {
			start := bytes.LastIndex(data[:offset], []byte{10}) + 1
			return bytes.Count(data[:start], []byte{10}) + 1, int(offset) - start
		}
		switch e := err.(type) {
		case *json.SyntaxError:
			line, col := getLineColFromOffset(data, e.Offset)
			return fmt.Errorf("%s:%d:%d: %v", filename, line, col, err)
		case *json.UnmarshalTypeError:
			line, col := getLineColFromOffset(data, e.Offset)
			return fmt.Errorf("%s:%d:%d: expected json %s value but got %s", filename, line, col, e.Type, e.Value)
		}
		return fmt.Errorf("%s: %v", filename, err)
	}
	return nil
}
