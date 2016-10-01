// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"resenje.org/antixsrf"
	"resenje.org/httphandlers"
	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/pkg/service"
	"gopherpit.com/gopherpit/services/user"
)

// A shorter variable for function that chains handlers.
var chainHandlers = httphandlers.ChainHandlers

// Helper function for raising unexpected errors in JSON API handlers.
func jsonServerError(w http.ResponseWriter, err error) {
	if _, ok := err.(net.Error); ok {
		jsonresponse.ServiceUnavailable(w, nil)
		return
	}
	jsonresponse.InternalServerError(w, nil)
}

// Helper function for raising unexpected errors in HTML frontend handlers.
func (s *Server) htmlServerError(w http.ResponseWriter, r *http.Request, err error) {
	if _, ok := err.(net.Error); ok {
		s.respondServiceUnavailable(w, r)
		return
	}
	s.respondInternalServerError(w, r)
}

// statusResponse is a response of a status API handler.
type statusResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
	service.Info
}

func (s Server) statusAPIHandler(w http.ResponseWriter, r *http.Request) {
	jsonresponse.OK(w, statusResponse{
		Name:    s.Name,
		Version: s.Version(),
		Uptime:  time.Since(s.startTime).String(),
		Info:    *service.NewInfo(),
	})
}

func (s Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s version %s, uptime %s", s.Name, s.Version(), time.Since(s.startTime))
}

// staticRecoveryHandler is a base hander for other recovery handlers.
func (s Server) staticRecoveryHandler(h http.Handler, body string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				debugInfo := fmt.Sprintf(
					"version: %s, build info: %s\r\n\r\n%s\r\n\r\n%#v\r\n\r\n%#v",
					s.Options.Version,
					s.BuildInfo,
					debug.Stack(),
					r.URL,
					r,
				)
				s.logger.Errorf("%v %v: %v\n%s", r.Method, r.URL.Path, err, debugInfo)
				go func() {
					defer s.RecoveryService.Recover()
					if err := s.EmailService.Notify(
						fmt.Sprint(
							"Panic ",
							r.Method,
							" ",
							r.URL.String(),
							": ", err,
						),
						debugInfo,
					); err != nil {
						s.logger.Error("panic handler email sending: ", err)
					}
				}()
				if contentType != "" {
					w.Header().Set("Content-Type", contentType)
				}
				w.WriteHeader(http.StatusInternalServerError)
				if body != "" {
					fmt.Fprintln(w, body)
				}
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// Recovery handler for HTML frontend routers.
func (s Server) htmlRecoveryHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				debugInfo := fmt.Sprintf(
					"version: %s, build info: %s\r\n\r\n%s\r\n\r\n%#v\r\n\r\n%#v",
					s.Options.Version,
					s.BuildInfo,
					debug.Stack(),
					r.URL,
					r,
				)
				s.logger.Errorf("%v %v: %v\n%s", r.Method, r.URL.Path, err, debugInfo)
				go func() {
					defer s.RecoveryService.Recover()
					if err := s.EmailService.Notify(
						fmt.Sprint(
							"Panic ",
							r.Method,
							" ",
							r.URL.String(),
							": ", err,
						),
						debugInfo,
					); err != nil {
						s.logger.Error("panic handler email sending: ", err)
					}
				}()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				s.htmlInternalServerErrorHandler(w, r)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// Recovery handler for JSON API routers.
func (s Server) jsonRecoveryHandler(h http.Handler) http.Handler {
	return s.staticRecoveryHandler(h, `{"message":"Server error","code": 500}`, "application/json; charset=utf-8")
}

// Recovery handler that does not write anything to response.
// It is useful as the firs handler in chain if a handler that
// transforms response needs to be before other recovery handler,
// like compress handler.
func (s Server) nilRecoveryHandler(h http.Handler) http.Handler {
	return s.staticRecoveryHandler(h, "", "")
}

func (s *Server) htmlMaxBodyBytesHandler(h http.Handler) http.Handler {
	m, err := renderToString(s.templateRequestEntityTooLarge(), "", nil)
	if err != nil {
		s.logger.Errorf("htmlMaxBodyBytesHandler Template413 error: %s", err)
		m = "Request Entity Too Large"
	}
	return httphandlers.MaxBodyBytesHandler(h, 20*1024, m, "text/html; charset=utf-8")
}

func textMaxBodyBytesHandler(h http.Handler) http.Handler {
	return httphandlers.MaxBodyBytesHandler(h, 20*1024, `Request Entity Too Large`, "text/plain; charset=utf-8")
}

func jsonMaxBodyBytesHandler(h http.Handler) http.Handler {
	return httphandlers.MaxBodyBytesHandler(h, 20*1024, `{"message":"Request Entity Too Large error","code":413}`, "application/json; charset=utf-8")
}

func (s Server) htmlNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.respondNotFound(w, r)
}

func jsonNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, `{"message": "Not found", "code": 404}`)
}

func (s Server) htmlForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	s.respondForbidden(w, r)
}

func (s Server) htmlInternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	s.respondInternalServerError(w, r)
}

func (s Server) generateAntiXSRFCookieHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(s.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		h.ServeHTTP(w, r)
	})
}

func (s Server) jsonAntiXSRFHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := antixsrf.Verify(r); err != nil {
			s.logger.Warningf("xsrf %s: %s", r.RequestURI, err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(w, `{"message": "Forbidden", "code": 403}`)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) htmlLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.user(r)
		if err != nil && err != user.UserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.logger.Errorf("email notify: %s", err)
				}
			}()
			s.htmlServerError(w, r, err)
			return
		}
		if r.Header.Get(s.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		if u == nil || u.Disabled {
			if r.Header.Get(s.SessionCookieName) != "" {
				s.logout(w, r)
			}
			respond(w, s.templateLogin(), nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) jsonLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.user(r)
		if err != nil && err != user.UserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.logger.Errorf("email notify: %s", err)
				}
			}()
			jsonServerError(w, err)
			return
		}
		if u == nil || u.Disabled {
			if r.Header.Get(s.SessionCookieName) != "" {
				s.logout(w, r)
			}
			jsonresponse.Unauthorized(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Handlers that acts as a splitter in a handler chain.
// If user is logged in, first argument handler will be executed,
// otherwise the second one will.
func (s *Server) htmlLoginAltHandler(h, alt http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.user(r)
		if err != nil && err != user.UserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.logger.Errorf("email notify: %s", err)
				}
			}()
			s.htmlServerError(w, r, err)
			return
		}
		if u == nil || u.Disabled {
			if r.Header.Get(s.SessionCookieName) != "" {
				s.logout(w, r)
			}
			alt.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) htmlValidatedEmailRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.user(r)
		if err != nil {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.logger.Errorf("email notify: %s", err)
				}
			}()
			s.htmlServerError(w, r, err)
			return
		}
		if u.EmailUnvalidated {
			respond(w, s.templateEmailUnvalidated(), map[string]interface{}{
				"User": u,
			})
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) jsonValidatedEmailRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.user(r)
		if err != nil && err != user.UserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.logger.Errorf("email notify: %s", err)
				}
			}()
			jsonServerError(w, err)
			return
		}
		if u.EmailUnvalidated {
			jsonresponse.Forbidden(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type textMethodHandler map[string]http.Handler

func (h textMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httphandlers.MethodHandler(h, "Method Not Allowed", "text/plain; charset=utf-8", w, r)
}

type jsonMethodHandler map[string]http.Handler

func (h jsonMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httphandlers.MethodHandler(h, `{"message": "Method Not Allowed", "code": 405}`, "application/json; charset=utf-8", w, r)
}

func noCacheHeaderHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

// Handler that allows requests made only from IPs form CIDRs defined in
// configuration as Internal CIDRs, to be used in HTML frontend routers.
func (s *Server) htmlIPAccessHandler(h http.Handler) http.Handler {
	return httphandlers.IPAccessHandler{
		Handler: h,
		UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.respondUnauthorized(w, r)
		}),
		CIDRs: &s.InternalCIDRs,
	}
}

// Handler that allows requests made only from IPs form CIDRs defined in
// configuration as Internal CIDRs, to be used in routers that returns
// plain text responses.
func (s Server) textIPAccessHandler(h http.Handler) http.Handler {
	return httphandlers.IPAccessHandler{
		Handler: h,
		UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Unauthorized")
		}),
		CIDRs: &s.InternalCIDRs,
	}
}

// Handler that allows requests made only from IPs form CIDRs defined in
// configuration as Internal CIDRs, to be used in JSON API routers.
func (s Server) jsonIPAccessHandler(h http.Handler) http.Handler {
	return httphandlers.IPAccessHandler{
		Handler: h,
		UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonresponse.Unauthorized(w, nil)
		}),
		CIDRs: &s.InternalCIDRs,
	}
}
