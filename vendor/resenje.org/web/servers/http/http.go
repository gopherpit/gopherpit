// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpServer

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Options struct holds parameters that can be configure using
// functions with prefix With.
type Options struct {
	tlsConfig *tls.Config
}

// Option is a function that sets optional parameters for
// the Server.
type Option func(*Options)

// WithTLSConfig sets a TLS configuration for the HTTP server
// and creates a TLS listener.
func WithTLSConfig(tlsConfig *tls.Config) Option { return func(o *Options) { o.tlsConfig = tlsConfig } }

// Server wraps http.Server to provide methods for
// resenje.org/web/servers.Server interface.
type Server struct {
	http.Server
}

// New creates a new instance of Server.
func New(handler http.Handler, opts ...Option) (s *Server) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	s = &Server{
		Server: http.Server{
			Handler:   handler,
			TLSConfig: o.tlsConfig,
		},
	}
	return
}

// Serve executes http.Server.Serve method.
// If the provided listener is net.TCPListener, keep alive
// will be enabled. If server is configured with TLS,
// a tls.Listener will be created with provided listener.
func (s *Server) Serve(ln net.Listener) (err error) {
	if l, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{TCPListener: l}
	}
	if s.TLSConfig != nil {
		ln = tls.NewListener(ln, s.TLSConfig)
	}

	err = s.Server.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return
}

// TCPKeepAliveListener sets TCP keep alive period.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept accepts TCP connection and sets TCP keep alive period
func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
