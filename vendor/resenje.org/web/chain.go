// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "net/http"

// ChainHandlers executes each function from the arguments with handler
// from the next function to construct a chan fo callers.
func ChainHandlers(handlers ...func(http.Handler) http.Handler) (h http.Handler) {
	for i := len(handlers) - 1; i >= 0; i-- {
		h = handlers[i](h)
	}
	return
}

// FinalHandler is a helper function to wrap the last http.Handler element
// in the ChainHandlers function.
func FinalHandler(h http.Handler) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return h
	}
}

// FinalHandlerFunc is a helper function to wrap the last function with signature
// func(w http.ResponseWriter, r *http.Request) in the ChainHandlers function.
func FinalHandlerFunc(h func(w http.ResponseWriter, r *http.Request)) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(h)
	}
}
