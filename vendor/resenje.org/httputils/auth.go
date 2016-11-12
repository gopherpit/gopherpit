// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const basicAuthScheme string = "Basic "

// AuthHandler is a net/http Handler that can be configured to check credentials from
// custom Key and Secret HTTP headers, or Basic auth from Authorization header.
// Depending on configuration of BasicAuthRealm, KeyHeaderName or SecretHeaderName,
// it can be used as"
//  - Basic auth handler - only BasicAuthRealm is set
//  - single API key auth handler - only KeyHeaderName is set
//  - single API key auth handler with Basic auth support - BasicAuthRealm and KeyHeaderName are set
//  - public/secret API key auth handler - KeyHeaderName and SecretHeaderName are set
//  - public/secret API key auth handler with Basic auth support - all three are set
// By setting AuthorizedNetworks, this handler can authorize requests based only on
// RemoteAddr address.
type AuthHandler struct {
	KeyHeaderName    string
	SecretHeaderName string
	BasicAuthRealm   string

	// Handler will be used if AuthFunc is successful.
	Handler http.Handler
	// UnauthorizedHandler will be used if AuthFunc is not successful.
	UnauthorizedHandler http.Handler
	// ErrorHandler will be used if there is an error. If it is nil, a panic will occur.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// AuthFunc validates credentials.
	AuthFunc func(r *http.Request, key, secret string) (valid bool, err error)
	// PostAuthFunc is a hook to log, set request context or preform any other action
	// after credentials check.
	PostAuthFunc func(w http.ResponseWriter, r *http.Request, key, secret string, valid bool) (rr *http.Request, err error)

	// AuthorizeAll will bypass all methods and authorize all requests.
	AuthorizeAll bool
	// AuthorizedNetworks are network ranges from where requests are authorized
	// without credentials check. Only address from request's RemoteAddr will be
	// checked.
	AuthorizedNetworks []net.IPNet
}

// ServeHTTP serves an HTTP response for a request.
func (h AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, secret, valid, err := h.authenticate(r)
	if err != nil {
		h.error(w, r, err)
		return
	}
	if h.PostAuthFunc != nil {
		rr, err := h.PostAuthFunc(w, r, key, secret, valid)
		if err != nil {
			h.error(w, r, err)
			return
		}
		if rr != nil {
			r = rr
		}
	}
	if !valid {
		h.unauthorized(w, r)
		return
	}

	if h.Handler != nil {
		h.Handler.ServeHTTP(w, r)
	}
}

func (h AuthHandler) authenticate(r *http.Request) (key, secret string, valid bool, err error) {
	if h.AuthorizeAll {
		valid = true
		return
	}

	if len(h.AuthorizedNetworks) > 0 {
		var host string
		host, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return
		}
		ip := net.ParseIP(host)
		for _, network := range h.AuthorizedNetworks {
			if network.Contains(ip) {
				valid = true
				return
			}
		}
	}

	if h.AuthFunc != nil {
		if h.KeyHeaderName != "" || h.SecretHeaderName != "" {
			if h.KeyHeaderName != "" {
				key = r.Header.Get(h.KeyHeaderName)
			}
			if h.SecretHeaderName != "" {
				secret = r.Header.Get(h.SecretHeaderName)
			}
			// Call AuthFunc and return only if there are provided data in headers.
			// If not, auth data from Authorization header should be validated.
			if key != "" || secret != "" {
				valid, err = h.AuthFunc(r, key, secret)
				return
			}
		}
		if h.BasicAuthRealm != "" {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, basicAuthScheme) {
				return
			}
			var decoded []byte
			decoded, err = base64.StdEncoding.DecodeString(auth[len(basicAuthScheme):])
			if err != nil {
				return
			}

			creds := bytes.SplitN(decoded, []byte(":"), 2)
			if len(creds) != 2 {
				return
			}
			key = string(creds[0])
			secret = string(creds[1])

			// This is the last auth method, so there is no need to check any values here,
			// they will be returned ath the and of a function.
			valid, err = h.AuthFunc(r, key, secret)
		}
	}

	return
}

func (h AuthHandler) error(w http.ResponseWriter, r *http.Request, err error) {
	if h.ErrorHandler == nil {
		panic(err)
	}
	h.ErrorHandler(w, r, err)
}

func (h AuthHandler) unauthorized(w http.ResponseWriter, r *http.Request) {
	if h.BasicAuthRealm != "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", h.BasicAuthRealm))
	}
	if h.UnauthorizedHandler != nil {
		h.UnauthorizedHandler.ServeHTTP(w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
