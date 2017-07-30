// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package servers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// Logger defines methods required for logging.
type Logger interface {
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// stdLogger is a simple implementation of Logger interface
// that uses log package for logging messages.
type stdLogger struct{}

func (l stdLogger) Infof(format string, a ...interface{}) {
	log.Printf("INFO "+format, a...)
}

func (l stdLogger) Errorf(format string, a ...interface{}) {
	log.Printf("ERROR "+format, a...)
}

// Option is a function that sets optional parameters for Servers.
type Option func(*Servers)

// WithLogger sets the Logger instance for logging messages.
func WithLogger(logger Logger) Option { return func(o *Servers) { o.logger = logger } }

// WithRecoverFunc sets a function that will be used to recover
// from panic inside a goroutune that servers are serving requests.
func WithRecoverFunc(recover func()) Option { return func(o *Servers) { o.recover = recover } }

// Servers holds a list of servers and their options.
// It provides a simple way to construct server group with Add method,
// to start them with Serve method, and stop them with Close or Shutdown methods.
type Servers struct {
	servers []*server
	mu      sync.Mutex
	logger  Logger
	recover func()
}

// New creates a new instance of Servers with applied options.
func New(opts ...Option) (s *Servers) {
	s = &Servers{
		logger:  stdLogger{},
		recover: func() {},
	}
	for _, opt := range opts {
		opt(s)
	}
	return
}

// Server defines required methods for a type that can be added to
// the Servers.
type Server interface {
	// Serve should start server responding to requests.
	// The listener is initialized and already listening.
	Serve(ln net.Listener) error
	// Close should stop server from serving all existing requests
	// and stop accepting new ones.
	// The listener provided in Serve method must stop listening.
	Close() error
	// Shutdown should gracefully stop server. All existing requests
	// should be processed within a deadline provided by the context.
	// No new requests should be accepted.
	// The listener provided in Serve method must stop listening.
	Shutdown(ctx context.Context) error
}

type server struct {
	Server
	name    string
	address string
	tcpAddr *net.TCPAddr
}

func (s *server) label() string {
	if s.name == "" {
		return "server"
	}
	return s.name + " server"
}

// Add adds a new server instance by a custom name and with
// address to listen to.
func (s *Servers) Add(name, address string, srv Server) {
	s.mu.Lock()
	s.servers = append(s.servers, &server{
		Server:  srv,
		name:    name,
		address: address,
	})
	s.mu.Unlock()
}

// Serve starts all added servers.
// New new servers must be added after this methid is called.
func (s *Servers) Serve() (err error) {
	lns := make([]net.Listener, len(s.servers))
	for i, srv := range s.servers {
		ln, err := net.Listen("tcp", srv.address)
		if err != nil {
			for _, l := range lns {
				if l == nil {
					continue
				}
				if err := l.Close(); err != nil {
					s.logger.Errorf("%s listener %q close: %v", srv.label(), srv.address, err)
				}
			}
			return fmt.Errorf("%s listener %q: %v", srv.label(), srv.address, err)
		}
		lns[i] = ln
	}
	for i, srv := range s.servers {
		go func(srv *server, ln net.Listener) {
			defer s.recover()

			s.mu.Lock()
			srv.tcpAddr = ln.Addr().(*net.TCPAddr)
			s.mu.Unlock()

			s.logger.Infof("%s listening on %q", srv.label(), srv.tcpAddr.String())
			if err = srv.Serve(ln); err != nil {
				s.logger.Errorf("%s serve %q: %v", srv.label(), srv.tcpAddr.String(), err)
			}
		}(srv, lns[i])
	}
	return nil
}

// Addr returns a TCP address of the listener that a server
// with a specific name is using. If there are more servers
// with the same name, the address of the first started server
// is returned.
func (s *Servers) Addr(name string) (a *net.TCPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, srv := range s.servers {
		if srv.name == name {
			return srv.tcpAddr
		}
	}
	return nil
}

// Close stops all servers, by calling Close method on each of them.
func (s *Servers) Close() {
	wg := &sync.WaitGroup{}
	for _, srv := range s.servers {
		wg.Add(1)
		go func(srv *server) {
			defer s.recover()
			defer wg.Done()

			s.logger.Infof("%s closing", srv.label())
			if err := srv.Close(); err != nil {
				s.logger.Errorf("%s close: %v", srv.label(), err)
			}
		}(srv)
	}
	wg.Wait()
	return
}

// Shutdown gracefully stops all servers, by calling Shutdown method on each of them.
func (s *Servers) Shutdown(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for _, srv := range s.servers {
		wg.Add(1)
		go func(srv *server) {
			defer s.recover()
			defer wg.Done()

			s.logger.Infof("%s shutting down", srv.label())
			if err := srv.Shutdown(ctx); err != nil {
				s.logger.Errorf("%s shutdown: %v", srv.label(), err)
			}
		}(srv)
	}
	wg.Wait()
	return
}
