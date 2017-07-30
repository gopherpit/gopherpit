// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"net/http"
)

// MaxBodyBytesHandler blocks requests with body size greater then specified, by
// responding with  a request entity too large 513 HTTP response.
// It is a wrapper around http.MaxBytesReader to also check Content-Length header
// and and multipart form requests.
type MaxBodyBytesHandler struct {
	// Handler will be used if limit is not reached.
	Handler http.Handler
	// Limit is a maximum number of bytes that a request body can have.
	Limit int64
	// BodyFunc response will be written as the response.
	BodyFunc func(r *http.Request) (string, error)
	// ContentType will be used as a value for
	ContentType string
	// ErrorHandler will be used if there is an error from BodyFunc. If it is nil,
	// a panic will occur.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

func (h MaxBodyBytesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > h.Limit {
		h.requestEntityTooLarge(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.Limit)

	if err := r.ParseMultipartForm(1024); r.Body == nil && err != nil {
		h.requestEntityTooLarge(w, r)
		return
	}
	h.Handler.ServeHTTP(w, r)
}

func (h MaxBodyBytesHandler) requestEntityTooLarge(w http.ResponseWriter, r *http.Request) {
	if h.ContentType != "" {
		w.Header().Set("Content-Type", h.ContentType)
	}
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	body, err := h.BodyFunc(r)
	if err != nil {
		if h.ErrorHandler == nil {
			panic(err)
		}
		h.ErrorHandler(w, r, err)
		return
	}
	fmt.Fprintln(w, body)
}
