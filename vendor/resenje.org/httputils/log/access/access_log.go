// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accessLog // import "resenje.org/httputils/log/access"

import (
	"net/http"
	"strings"
	"time"

	"resenje.org/logging"
)

type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) CloseNotify() <-chan bool {
	return l.w.(http.CloseNotifier).CloseNotify()
}

func (l *responseLogger) Flush() {
	l.w.(http.Flusher).Flush()
}

func (l *responseLogger) Write(b []byte) (int, error) {
	if l.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		l.status = http.StatusOK
	}
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *responseLogger) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

// NewHandler returns a handler that logs HTTP requests.
// It logs information about remote address, X-Forwarded-For or X-Real-Ip,
// HTTP method, request URI, HTTP protocol, HTTP response status, total bytes
// written to http.ResponseWriter, response duration, HTTP referrer and
// HTTP client user agent.
func NewHandler(h http.Handler, logger *logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		rl := &responseLogger{w, 0, 0}
		h.ServeHTTP(rl, r)
		referrer := r.Referer()
		if referrer == "" {
			referrer = "-"
		}
		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = "-"
		}
		ips := []string{}
		xfr := r.Header.Get("X-Forwarded-For")
		if xfr != "" {
			ips = append(ips, xfr)
		}
		xri := r.Header.Get("X-Real-Ip")
		if xri != "" {
			ips = append(ips, xri)
		}
		xips := "-"
		if len(ips) > 0 {
			xips = strings.Join(ips, ", ")
		}
		var level logging.Level
		switch {
		case rl.status >= 500:
			level = logging.ERROR
		case rl.status >= 400:
			level = logging.WARNING
		case rl.status >= 300:
			level = logging.INFO
		case rl.status >= 200:
			level = logging.INFO
		default:
			level = logging.DEBUG
		}
		logger.Logf(level, "%s \"%s\" \"%v %s %v\" %d %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, r.RequestURI, r.Proto, rl.status, rl.size, time.Since(startTime).Seconds(), referrer, userAgent)
	})
}
