package httphandlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

const basicAuthScheme string = "Basic "

type BasicAuthHandler struct {
	Handler             http.Handler
	UnauthorizedHandler http.Handler
	ErrorHandler        func(w http.ResponseWriter, r *http.Request, err error)
	ValidateFunc        func(r *http.Request, username, password string) (rr *http.Request, valid bool, err error)
	AuthorizeAll        bool
	AuthorizedNetworks  []net.IPNet
	BasicAuthRealm      string
	Logger              *log.Logger
}

func (h BasicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if h.AuthorizeAll {
		h.Handler.ServeHTTP(w, r)
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
				h.Handler.ServeHTTP(w, r)
				return
			}
		}
	}

	if h.BasicAuthRealm != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, basicAuthScheme) {
			h.unauthorized(w, r)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(auth[len(basicAuthScheme):])
		if err != nil {
			h.unauthorized(w, r)
			return
		}

		creds := bytes.SplitN(decoded, []byte(":"), 2)
		if len(creds) != 2 {
			h.unauthorized(w, r)
			return
		}

		if h.ValidateFunc != nil {
			r, valid, err := h.ValidateFunc(r, string(creds[0]), string(creds[1]))
			if err != nil {
				if h.Logger != nil {
					h.Logger.Printf("basic auth: %s", err)
				}
				if h.ErrorHandler != nil {
					h.ErrorHandler(w, r, err)
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if !valid {
				h.unauthorized(w, r)
				return
			}
		}
	}

	h.Handler.ServeHTTP(w, r)
}

func (h BasicAuthHandler) unauthorized(w http.ResponseWriter, r *http.Request) {
	if h.BasicAuthRealm != "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", h.BasicAuthRealm))
	}
	if h.UnauthorizedHandler != nil {
		h.UnauthorizedHandler.ServeHTTP(w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
