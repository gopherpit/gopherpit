// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net"
	"net/http"
	"strings"
)

// GetRequestIPs returns all possible IPs found in HTTP request.
func GetRequestIPs(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	ips := []string{ip}
	xfr := r.Header.Get("X-Forwarded-For")
	if xfr != "" {
		ips = append(ips, xfr)
	}
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		ips = append(ips, xri)
	}
	return strings.Join(ips, ", ")
}

// GetRequestEndpoint returns request's host perpended with protocol:
// protocol://host.
func GetRequestEndpoint(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS == nil {
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + r.Host
}
