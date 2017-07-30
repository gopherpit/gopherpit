// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer

import (
	"errors"
	"net/http"
)

// Default file handlers in case of errors.
var (
	DefaultNotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})
	DefaultForbiddenHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
	DefaultInternalServerErrorHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	errNotFound       = errors.New("not found")
	errNotRegularFile = errors.New("not a regular file")
)

// Options contains parameters for file server configuration.
type Options struct {
	// If Hasher is not nil, it will be used to compute a file hash.
	Hasher Hasher
	// Do not hash file paths if the request contains query parameters.
	// It may be used if other tools construct a request with predefined
	// bust parameter.
	NoHashQueryStrings bool
	// Redirect trailing slash of HTTP requests.
	RedirectTrailingSlash bool
	// IndexPage is a filename that will be used to render an index page
	// for a directory.
	IndexPage string
	// AltDir is a directory path to look for files before the
	// initialized directory.
	AltDir string

	// NotFoundHandler is used when no file can be found.
	NotFoundHandler http.Handler
	// ForbiddenHandler is used when permissions to open file are insufficient.
	ForbiddenHandler http.Handler
	// InternalServerErrorHandler is used when an unexpected error occurs.
	InternalServerErrorHandler http.Handler
}
