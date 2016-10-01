package httphandlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type KeyAuth struct {
	Handler             http.Handler
	UnauthorizedHandler http.Handler
	Keys                map[string]bool
	ValidateFunc        func(key string) bool
	PostAuthFunc        func(key string, valid bool, w http.ResponseWriter, r *http.Request) bool
	AuthorizeAll        bool
	AuthorizedNetworks  []net.IPNet
	HeaderName          string
	BasicAuthRealm      string
}

func (h KeyAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.UnauthorizedHandler == nil {
		h.UnauthorizedHandler = http.HandlerFunc(defaultKeyAuthUnauthorizedHandler)
	}

	key, valid := h.authenticate(r)

	if h.PostAuthFunc != nil && h.PostAuthFunc(key, valid, w, r) == false {
		return
	}

	if !valid {
		h.unauthorized(w, r)
		return
	}

	h.Handler.ServeHTTP(w, r)
}

func (h KeyAuth) authenticate(r *http.Request) (key string, valid bool) {
	if h.AuthorizeAll {
		valid = true
		return
	}

	if len(h.AuthorizedNetworks) > 0 {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			panic(err)
		}
		ip := net.ParseIP(host)
		for _, network := range h.AuthorizedNetworks {
			if network.Contains(ip) {
				valid = true
				return
			}
		}
	}

	if h.HeaderName != "" {
		key = r.Header.Get(h.HeaderName)
		if key != "" {
			if enabled, ok := h.Keys[key]; ok {
				valid = enabled
				return
			}
			if h.ValidateFunc != nil {
				valid = h.ValidateFunc(key)
				return
			}
		}
	}

	if h.BasicAuthRealm != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, basicAuthScheme) {
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(auth[len(basicAuthScheme):])
		if err != nil {
			return
		}

		creds := bytes.SplitN(decoded, []byte(":"), 2)
		if len(creds) != 2 {
			return
		}

		key = string(creds[0])
		if key == "" {
			key = string(creds[1])
		}
		if key != "" {
			if enabled, ok := h.Keys[key]; ok {
				valid = enabled
				return
			}
			if h.ValidateFunc != nil {
				valid = h.ValidateFunc(key)
				return
			}
		}
	}

	return
}

func (h KeyAuth) unauthorized(w http.ResponseWriter, r *http.Request) {
	if h.BasicAuthRealm != "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", h.BasicAuthRealm))
	}
	h.UnauthorizedHandler.ServeHTTP(w, r)
}

func defaultKeyAuthUnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
