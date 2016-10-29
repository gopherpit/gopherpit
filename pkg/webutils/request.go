// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webutils

import (
	"net/http"
	"strings"
)

// GetIPs returns all possible IPs found in HTTP request.
func GetIPs(r *http.Request) string {
	var ips []string
	idx := strings.LastIndex(r.RemoteAddr, ":")
	if idx == -1 {
		ips = []string{r.RemoteAddr}
	} else {
		ips = []string{r.RemoteAddr[:idx]}
	}
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		ips = append(ips, xri)
	}
	xfr := r.Header.Get("X-Forwarded-For")
	if xfr != "" {
		ips = append(ips, xfr)
	}
	return strings.Join(ips, ", ")
}

// GetHost returns request's host perpended with protocol: protocol://host.
func GetHost(r *http.Request) string {
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
