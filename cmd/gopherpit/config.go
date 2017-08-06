// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"resenje.org/x/application"

	"gopherpit.com/gopherpit/server/config"
)

var (
	// Initialize configurations with default values.
	options            = config.NewGopherPitOptions()
	apiOptions         = config.NewAPIOptions()
	loggingOptions     = config.NewLoggingOptions()
	emailOptions       = config.NewEmailOptions()
	ldapOptions        = config.NewLDAPOptions()
	sessionOptions     = config.NewSessionOptions()
	userOptions        = config.NewUserOptions()
	certificateOptions = config.NewCertificateOptions()
	servicesOptions    = config.NewServicesOptions()
)

func init() {
	// Register options in config.
	cfg.Register(config.Name, options)
	cfg.Register("api", apiOptions)
	cfg.Register("logging", loggingOptions)
	cfg.Register("email", emailOptions)
	cfg.Register("ldap", ldapOptions)
	cfg.Register("session", sessionOptions)
	cfg.Register("user", userOptions)
	cfg.Register("certificate", certificateOptions)
	cfg.Register("services", servicesOptions)
}

var cfg = application.NewConfig(config.Name)

func configCmd() {
	// Print loaded configuration.
	fmt.Print(cfg.String())
}

func updateConfig() {
	if *configDir == "" {
		*configDir = os.Getenv(strings.ToUpper(config.Name) + "_CONFIGDIR")
	}
	if *configDir == "" {
		*configDir = config.Dir
	}

	cfg.Dirs = []string{
		filepath.Join(config.BaseDir, "defaults"),
		*configDir,
	}
	if err := cfg.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(2)
	}
}

func verifyAndPrepareConfig() {
	if err := cfg.VerifyAndPrepare(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if e, ok := err.(*application.HelpError); ok {
			fmt.Println()
			fmt.Println(e.Help)
		}
		os.Exit(2)
	}
}
