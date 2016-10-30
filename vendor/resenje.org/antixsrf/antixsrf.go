// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package antixsrf // import "resenje.org/antixsrf"

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	// XSRFCookieName is an HTTP cookie name to store anti-XSRF token.
	XSRFCookieName = "secid"
	// XSRFHeaderName is an HTTP header name to check the token.
	XSRFHeaderName = "X-Secid"
	// XSRFFormFieldName is an HTTP form field name to check the token.
	XSRFFormFieldName = "secid"
)

// Error is a generic error for this package.
type Error struct {
	message string
}

func newError(message string) *Error {
	return &Error{
		message: message,
	}
}

func (e *Error) Error() string {
	return e.message
}

// Errors related to invalid or missing anti-XSRF token value.
var (
	ErrNoReferer      = newError("antixsrf: missing referer header")
	ErrInvalidReferer = newError("antixsrf: invalid referer header")
	ErrInvalidToken   = newError("antixsrf: invalid xsrf token")
	ErrMissingCookie  = newError("antixsrf: missing xsrf cookie")
	ErrMissingToken   = newError("antixsrf: missing xsrf token")
	ErrMissingHeader  = newError("antixsrf: missing xsrf header")
)

var safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

// Verify check for a valid token in request Cookie, form field or header.
// It also checks if header "Referer" is present and that host values of
// the request and referrer are the same
func Verify(r *http.Request) error {
	if contains(safeMethods, r.Method) {
		return nil
	}

	referer, err := url.Parse(r.Header.Get("Referer"))
	if err != nil {
		return err
	}
	if referer.String() == "" {
		return ErrNoReferer
	}

	if !(r.Host == referer.Host) {
		return ErrInvalidReferer
	}

	token, err := r.Cookie(XSRFCookieName)
	if err != nil {
		if err.Error() == "http: named cookie not present" {
			return ErrMissingCookie
		}
		return err
	}

	if contains([]string{"application/x-www-form-urlencoded", "multipart/form-data"}, r.Header.Get("Content-Type")) {
		if r.FormValue(XSRFFormFieldName) == token.Value {
			return nil
		}
	}

	if token.Value == "" {
		return ErrMissingToken
	}

	header := r.Header.Get(XSRFHeaderName)

	if header == "" {
		return ErrMissingHeader
	}

	if header == token.Value {
		return nil
	}

	return ErrInvalidToken
}

// Generate generates an anti-XSRF token and sets it as a cookie value.
func Generate(w http.ResponseWriter, r *http.Request, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:   XSRFCookieName,
		Value:  newKey(),
		Path:   path,
		Secure: r.TLS != nil,
	})
}

func newKey() string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString(generateRandomKey(16)), "=")
}

func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
