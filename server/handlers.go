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
	"resenje.org/httputils"
	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/pkg/info"
	"gopherpit.com/gopherpit/services/user"
)

var (
	// Shorter variables for functions that chain handlers.
	chainHandlers    = httputils.ChainHandlers
	finalHandler     = httputils.FinalHandler
	finalHandlerFunc = httputils.FinalHandlerFunc
)

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
		s.htmlServiceUnavailableHandler(w, r)
		return
	}
	s.htmlInternalServerErrorHandler(w, r)
}

func textServerError(w http.ResponseWriter, err error) {
	if _, ok := err.(net.Error); ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, http.StatusText(http.StatusServiceUnavailable))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, http.StatusText(http.StatusInternalServerError))
}

// statusResponse is a response of a status API handler.
type statusResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
	info.Information
}

func (s Server) statusAPIHandler(w http.ResponseWriter, r *http.Request) {
	jsonresponse.OK(w, statusResponse{
		Name:        s.Name,
		Version:     s.Version(),
		Uptime:      time.Since(s.startTime).String(),
		Information: *info.NewInformation(),
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
	return s.staticRecoveryHandler(h, `{"code":500,"message":"Server error"}`, "application/json; charset=utf-8")
}

// Recovery handler that does not write anything to response.
// It is useful as the firs handler in chain if a handler that
// transforms response needs to be before other recovery handler,
// like compress handler.
func (s Server) nilRecoveryHandler(h http.Handler) http.Handler {
	return s.staticRecoveryHandler(h, "", "")
}

func (s *Server) htmlMaxBodyBytesHandler(h http.Handler) http.Handler {
	return httputils.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return renderToString(s.templates["RequestEntityTooLarge"], "", nil)
		},
		ContentType:  "text/html; charset=utf-8",
		ErrorHandler: s.htmlServerError,
	}
}

func textMaxBodyBytesHandler(h http.Handler) http.Handler {
	return httputils.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return `Request Entity Too Large`, nil
		},
		ContentType:  "text/plain; charset=utf-8",
		ErrorHandler: nil,
	}
}

func jsonMaxBodyBytesHandler(h http.Handler) http.Handler {
	return httputils.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return `{"code":413,"message":"Request Entity Too Large error"}`, nil
		},
		ContentType:  "application/json; charset=utf-8",
		ErrorHandler: nil,
	}
}

func textNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
}

func jsonNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	jsonresponse.NotFound(w, nil)
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
			jsonresponse.Forbidden(w, nil)
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
			s.respond(w, "Login", nil)
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
			s.respond(w, "EmailUnvalidated", map[string]interface{}{
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
	httputils.HandleMethods(h, "Method Not Allowed", "text/plain; charset=utf-8", w, r)
}

type jsonMethodHandler map[string]http.Handler

func (h jsonMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httputils.HandleMethods(h, `{"code":405,"message":"Method Not Allowed"}`, "application/json; charset=utf-8", w, r)
}

func noCacheHeaderHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}
