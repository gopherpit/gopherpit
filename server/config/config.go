// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config holds project and service related data and structures
// that define optional parameters for different parts of the service.
package config // import "gopherpit.com/gopherpit/server/config"

import (
	"fmt"
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
	// Set the version on build with: go build -ldflags "-X gopherpit.com/gopherpit/server/config.BuildInfo=$(shell git describe --long --dirty --always)"
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
		path, err := os.Executable()
		if err != nil {
			panic(err)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			panic(err)
		}
		return filepath.Dir(path)
	}()

	// Dir is default directory where configuration files are located.
	// Set the version on build with: go build -ldflags "-X gopherpit.com/gopherpit/server/config.Dir=$(CONFIG_DIR)"
	Dir = "/etc/gopherpit"
)
