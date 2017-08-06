// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package maintenance

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Values for HTTP Contet-Type header.
var (
	HTMLContentType = "text/html; charset=utf-8"
	TextContentType = "text/text; charset=utf-8"
	JSONContentType = "application/json; charset=utf-8"
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

// Store defines methods that are reqired to check, set and remove
// information wheather the maintenance is on of off.
// Usually only one boolean value is needed to be stored
type Store interface {
	// Return true if maintenance is enabled.
	Status() (on bool, err error)
	// Enable maintenance and returns true if the state has changed.
	On() (changed bool, err error)
	// Disables maintenance and returns true if the state has changed.
	Off() (changed bool, err error)
}

// MemoryStore implements Store that keeps data in memory.
type MemoryStore struct {
	on bool
	mu sync.Mutex
}

// NewMemoryStore creates a new instance of MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Status returns true if maintenance is enabled.
func (s *MemoryStore) Status() (on bool, err error) {
	s.mu.Lock()
	on = s.on
	s.mu.Unlock()
	return
}

// On enables maintenance.
func (s *MemoryStore) On() (changed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.on {
		return
	}
	s.on = true
	changed = true
	return
}

// Off disables maintenance.
func (s *MemoryStore) Off() (changed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.on {
		return
	}
	s.on = false
	changed = true
	return
}

// FileStore implements Store that manages maintenance
// status by existence of a specific file. If file exists
// maintenance is enabled, otherwise is disabled.
// This store persists maintenance state and provides
// a simple way to set maintenance on local filesystem
// with external tools.
type FileStore struct {
	filename string
}

// NewFileStore creates a new instance of FileStore.
func NewFileStore(filename string) *FileStore {
	return &FileStore{
		filename: filename,
	}
}

// Status returns true if maintenance is enabled.
func (s *FileStore) Status() (on bool, err error) {
	_, err = os.Stat(s.filename)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// On enables maintenance.
func (s *FileStore) On() (changed bool, err error) {
	if _, err = os.Stat(s.filename); err == nil {
		return
	}
	err = os.MkdirAll(filepath.Dir(s.filename), 0777)
	if err != nil {
		return
	}
	f, err := os.Create(s.filename)
	if err != nil {
		return
	}
	f.Close()
	changed = true
	return
}

// Off disables maintenance.
func (s *FileStore) Off() (changed bool, err error) {
	if _, err = os.Stat(s.filename); os.IsNotExist(err) {
		return
	}
	if err = os.Remove(s.filename); err != nil {
		return
	}
	changed = true
	return
}

// Response holds configuration for HTTP response
// during maintenance mode.
type Response struct {
	// Body will be returned if Handler is nil.
	Body    string
	Handler http.Handler
}

// Service implements http.Service interface to write a custom
// HTTP response during maintenance mode.
// It also provides JSON API handlers that can be used to
// check, set and remove maintenance mode.
type Service struct {
	HTML Response
	JSON Response
	Text Response

	store  Store
	logger Logger
}

// Option is a function that sets optional parameters to the Handler.
type Option func(*Service)

// WithStore sets Store to the Handler. If this option
// is not used, handler defaults to MemoryStore.
func WithStore(store Store) Option { return func(o *Service) { o.store = store } }

// WithLogger sets the function that will perform message logging.
// Default is log.Printf.
func WithLogger(logger Logger) Option { return func(o *Service) { o.logger = logger } }

// New creates a new instance of Handler.
// The first argument is the handler that will be executed
// when maintenance mode is off.
func New(options ...Option) (s *Service) {
	s = &Service{
		logger: stdLogger{},
	}
	for _, option := range options {
		option(s)
	}
	if s.store == nil {
		s.store = NewMemoryStore()
	}
	return
}

// HTMLHandler is a HTTP middleware that should be used
// alongide HTML pages.
func (s Service) HTMLHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		on, err := s.store.Status()
		if err != nil {
			s.logger.Errorf("maintenance status: %v", err)
		}
		if on || err != nil {
			if s.HTML.Handler != nil {
				s.HTML.Handler.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", HTMLContentType)
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, s.HTML.Body)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// JSONHandler is a HTTP middleware that should be used
// alongide JSON-encoded responses.
func (s Service) JSONHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		on, err := s.store.Status()
		if err != nil {
			s.logger.Errorf("maintenance status: %v", err)
		}
		if on || err != nil {
			if s.JSON.Handler != nil {
				s.JSON.Handler.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", JSONContentType)
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, s.JSON.Body)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// TextHandler is a HTTP middleware that should be used
// alongide plaintext responses.
func (s Service) TextHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		on, err := s.store.Status()
		if err != nil {
			s.logger.Errorf("maintenance status: %v", err)
		}
		if on || err != nil {
			if s.Text.Handler != nil {
				s.Text.Handler.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", TextContentType)
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, s.Text.Body)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// StatusHandler can be used in JSON-encoded HTTP API
// to check the status of maintenance.
func (s Service) StatusHandler(w http.ResponseWriter, r *http.Request) {
	on, err := s.store.Status()
	if err != nil {
		s.logger.Errorf("maintenance status: %s", err)
		jsonInternalServerErrorResponse(w)
		return
	}
	jsonStatusResponse(w, on)
}

// OnHandler can be used in JSON-encoded HTTP API to enable maintenance.
// It returns HTTP Status Created if the maintenance is enabled.
// If the maintenance is already enabled, it returns HTTP Status OK.
func (s Service) OnHandler(w http.ResponseWriter, r *http.Request) {
	changed, err := s.store.On()
	if err != nil {
		s.logger.Errorf("maintenance on: %s", err)
		jsonInternalServerErrorResponse(w)
		return
	}
	if changed {
		s.logger.Infof("maintenance on")
		jsonCreatedResponse(w)
		return
	}
	jsonOKResponse(w)
}

// OffHandler can be used in JSON-encoded HTTP API to disable maintenance.
func (s Service) OffHandler(w http.ResponseWriter, r *http.Request) {
	changed, err := s.store.Off()
	if err != nil {
		s.logger.Errorf("maintenance off: %s", err)
		jsonInternalServerErrorResponse(w)
		return
	}
	if changed {
		s.logger.Infof("maintenance off")
	}
	jsonOKResponse(w)
}

func jsonOKResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"message":"OK","code":200}`)
}

func jsonCreatedResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, `{"message":"Created","code":201}`)
}

func jsonInternalServerErrorResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, `{"message":"Internal Server Error","code":500}`)
}

func jsonStatusResponse(w http.ResponseWriter, on bool) {
	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(http.StatusOK)
	if on {
		fmt.Fprintln(w, `{"status":"on"}`)
	} else {
		fmt.Fprintln(w, `{"status":"off"}`)
	}
}
