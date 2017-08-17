// Copyright (c) 2015-2017 Janoš Guljaš <janos@resenje.org>
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

// VerifyOptions holds optional parameters for the Generate function.
type VerifyOptions struct {
	cookieName    string
	headerName    string
	formFieldName string
}

// VerifyOption sets parameters defined in VerifyOptions.
type VerifyOption func(*VerifyOptions)

// WithVerifyCookieName sets the cookie name to check the token.
// Default is "secid".
func WithVerifyCookieName(name string) VerifyOption {
	return func(o *VerifyOptions) { o.cookieName = name }
}

// WithVerifyHeaderName sets the HTTP header name to check the token.
// Default is "X-Secid".
func WithVerifyHeaderName(name string) VerifyOption {
	return func(o *VerifyOptions) { o.headerName = name }
}

// WithVerifyFormFieldName sets the HTTP form field name to check the token.
// Default is "secid".
func WithVerifyFormFieldName(name string) VerifyOption {
	return func(o *VerifyOptions) { o.formFieldName = name }
}

// Verify check for a valid token in request Cookie, form field or header.
// It also checks if header "Referer" is present and that host values of
// the request and referrer are the same
func Verify(r *http.Request, opts ...VerifyOption) error {
	o := &VerifyOptions{
		cookieName:    XSRFCookieName,
		headerName:    XSRFHeaderName,
		formFieldName: XSRFFormFieldName,
	}
	for _, opt := range opts {
		opt(o)
	}

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

	token, err := r.Cookie(o.cookieName)
	if err != nil {
		if err.Error() == "http: named cookie not present" {
			return ErrMissingCookie
		}
		return err
	}

	if contains([]string{"application/x-www-form-urlencoded", "multipart/form-data"}, r.Header.Get("Content-Type")) {
		if r.FormValue(o.formFieldName) == token.Value {
			return nil
		}
	}

	if token.Value == "" {
		return ErrMissingToken
	}

	header := r.Header.Get(o.headerName)

	if header == "" {
		return ErrMissingHeader
	}

	if header == token.Value {
		return nil
	}

	return ErrInvalidToken
}

// GenerateOptions holds optional parameters for the Generate function.
type GenerateOptions struct {
	name   string
	path   string
	maxAge int
	force  bool
}

// GenerateOption sets parameters defined in GenerateOptions.
type GenerateOption func(*GenerateOptions)

// WithGenerateCookieName sets the cookie name that will be generated.
// Default is "secid".
func WithGenerateCookieName(name string) GenerateOption {
	return func(o *GenerateOptions) { o.name = name }
}

// WithGenerateCookiePath sets the cookie path. Default is "/".
func WithGenerateCookiePath(path string) GenerateOption {
	return func(o *GenerateOptions) { o.path = path }
}

// WithGenerateCookieMaxAge sets the cookie max age value in seconds.
// Default is 0 which sets a session lived cookie.
func WithGenerateCookieMaxAge(maxAge int) GenerateOption {
	return func(o *GenerateOptions) { o.maxAge = maxAge }
}

// WithGenerateForce sets the new cookie with token even if it exists.
// Default is false.
func WithGenerateForce(force bool) GenerateOption {
	return func(o *GenerateOptions) { o.force = force }
}

// Generate generates an anti-XSRF token and sets it as a cookie value.
func Generate(w http.ResponseWriter, r *http.Request, opts ...GenerateOption) {
	o := &GenerateOptions{
		name:   XSRFCookieName,
		path:   "/",
		maxAge: 0,
	}
	for _, opt := range opts {
		opt(o)
	}
	if !hasCookie(r, o.name) || o.force {
		http.SetCookie(w, &http.Cookie{
			Name:   o.name,
			Value:  newKey(),
			Path:   o.path,
			Secure: r.TLS != nil,
			MaxAge: o.maxAge,
		})
	}
}

func hasCookie(r *http.Request, name string) (yes bool) {
	_, err := r.Cookie(name)
	return err != http.ErrNoCookie
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

// GenerateHandler is a helper function that generates anti-XSRF cookie
// with default options inside a http handler middleware that can be chained
// with other http handlers.
func GenerateHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Generate(w, r)
		h.ServeHTTP(w, r)
	})
}
