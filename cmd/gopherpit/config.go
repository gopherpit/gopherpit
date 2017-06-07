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
	// Make options list to be able to use them in config.Update and
	// config.Prepare.
	allOptions = []config.Options{
		options,
		apiOptions,
		loggingOptions,
		emailOptions,
		ldapOptions,
		sessionOptions,
		userOptions,
		certificateOptions,
		servicesOptions,
	}
)

func configCmd() {
	// Print loaded configuration.
	fmt.Printf("# %s\n---\n%s\n", config.Name, options.String())
	fmt.Printf("# logging\n---\n%s\n", loggingOptions.String())
	fmt.Printf("# email\n---\n%s\n", emailOptions.String())
	fmt.Printf("# ldap\n---\n%s\n", ldapOptions.String())
	fmt.Printf("# session\n---\n%s\n", sessionOptions.String())
	fmt.Printf("# user\n---\n%s\n", userOptions.String())
	fmt.Printf("# certificate\n---\n%s\n", certificateOptions.String())
	fmt.Printf("# api\n---\n%s\n", apiOptions.String())
	fmt.Printf("# services\n---\n%s\n", servicesOptions.String())
	fmt.Printf("# config directories\n---\n- %s\n- %s\n", *configDir, filepath.Join(config.BaseDir, "defaults"))
}

func updateConfig() {
	if *configDir == "" {
		*configDir = os.Getenv(strings.ToUpper(config.Name) + "_CONFIGDIR")
	}
	if *configDir == "" {
		*configDir = config.Dir
	}
	// Update options structures based on files in configDir and environment
	// variables.
	if err := config.Update(allOptions, filepath.Join(config.BaseDir, "defaults"), *configDir); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}

func verifyAndPrepareConfig() {
	// Verify options values and provide help and error message in case of
	// an error.
	if help, err := config.Verify(allOptions); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if help != "" {
			fmt.Println()
			fmt.Println(help)
		}
		os.Exit(2)
	}
	// Execute prepare methods on options structures.
	// Usually it creates required directories or files.
	if err := config.Prepare(allOptions); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}
