// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

// DomainRedirectHandler responds with redirect url based on
// domain and httpsPort, othervise it executes the handler.
func DomainRedirectHandler(h http.Handler, domain, httpsPort string) http.Handler {
	if domain == "" && httpsPort == "" {
		return h
	}

	scheme := "http"
	port := ""
	if httpsPort != "" {
		if _, err := strconv.Atoi(httpsPort); err == nil {
			scheme = "https"
			port = httpsPort
		}
		if _, p, err := net.SplitHostPort(httpsPort); err == nil {
			scheme = "https"
			port = p
		}
	}
	if port == "443" {
		port = ""
	}
	var altDomain string
	if strings.HasPrefix("www.", domain) {
		altDomain = strings.TrimPrefix(domain, "www.")
	} else {
		altDomain = "www." + domain
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d, p, err := net.SplitHostPort(r.Host)
		if err != nil {
			d = r.Host
		}
		rs := r.URL.Scheme
		if fs := r.Header.Get("X-Forwarded-Proto"); fs != "" {
			rs = strings.ToLower(fs)
		}
		s := scheme
		if rs == "https" {
			s = "https"
		}
		if d == domain && rs == s {
			h.ServeHTTP(w, r)
			return
		}
		switch {
		case s == "http" && p == "80":
			p = ""
		case s == "https" && p == "443":
			p = ""
		case port != "":
			p = ":" + port
		case p != "":
			p = ":" + p
		}
		if d == altDomain {
			http.Redirect(w, r, strings.Join([]string{s, "://", domain, p, r.RequestURI}, ""), http.StatusMovedPermanently)
			return
		}
		http.Redirect(w, r, strings.Join([]string{s, "://", domain, p, r.RequestURI}, ""), http.StatusFound)
	})
}
