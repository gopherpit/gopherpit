// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"resenje.org/antixsrf"
	"resenje.org/jsonresponse"
	"resenje.org/web"
	"resenje.org/web/log/access"
	"resenje.org/web/recovery"

	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/user"
)

// Helper function for raising unexpected errors in JSON API handlers.
func jsonServerError(w http.ResponseWriter, err error) {
	if _, ok := err.(net.Error); ok {
		jsonresponse.ServiceUnavailable(w, nil)
		return
	}
	jsonresponse.InternalServerError(w, nil)
}

func (s *Server) htmlError(w http.ResponseWriter, r *http.Request, c int) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		s.Logger.Errorf("get user: %s", err)
		if _, ok := err.(net.Error); ok {
			s.html.RespondWithStatus(w, http.StatusText(http.StatusServiceUnavailable), nil, http.StatusServiceUnavailable)
			return
		}
		s.html.RespondWithStatus(w, http.StatusText(http.StatusInternalServerError), nil, http.StatusInternalServerError)
		return
	}
	if u == nil {
		s.html.RespondWithStatus(w, http.StatusText(c), nil, c)
		return
	}
	s.html.RespondWithStatus(w, http.StatusText(c)+" Private", map[string]interface{}{
		"User": u,
	}, c)
}

func (s *Server) htmlServerError(w http.ResponseWriter, r *http.Request, err error) {
	if _, ok := err.(net.Error); ok {
		s.htmlError(w, r, http.StatusServiceUnavailable)
		return
	}
	s.htmlError(w, r, http.StatusInternalServerError)
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

func (s *Server) htmlNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.htmlError(w, r, http.StatusNotFound)
}

func (s *Server) htmlForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	s.htmlError(w, r, http.StatusForbidden)
}

func (s *Server) htmlInternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	s.htmlError(w, r, http.StatusInternalServerError)
}

// statusResponse is a response of a status API handler.
type statusResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

func (s *Server) statusAPIHandler(w http.ResponseWriter, r *http.Request) {
	jsonresponse.OK(w, statusResponse{
		Name:    s.Name,
		Version: s.version(),
		Uptime:  time.Since(s.startTime).String(),
	})
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s version %s, uptime %s", s.Name, s.version(), time.Since(s.startTime))
}

func (s *Server) accessLogHandler(h http.Handler) http.Handler {
	return accessLog.NewHandler(h, s.AccessLogger)
}

// Recovery handler for HTML frontend routers.
func (s *Server) htmlRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(s.version()),
		recovery.WithLogFunc(s.Logger.Errorf),
		recovery.WithNotifier(s.EmailService),
		recovery.WithPanicResponseHandler(http.HandlerFunc(s.htmlInternalServerErrorHandler)),
	)
}

// Recovery handler for JSON API routers.
func (s *Server) jsonRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(s.version()),
		recovery.WithLogFunc(s.Logger.Errorf),
		recovery.WithNotifier(s.EmailService),
		recovery.WithPanicResponse(`{"message":"Internal Server Error","code":500}`, "application/json; charset=utf-8"),
	)
}

// Recovery handler that does not write anything to response.
// It is useful as the firs handler in chain if a handler that
// transforms response needs to be before other recovery handler,
// like compress handler.
func (s *Server) nilRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(s.version()),
		recovery.WithLogFunc(s.Logger.Errorf),
		recovery.WithNotifier(s.EmailService),
	)
}

func (s *Server) htmlMaxBodyBytesHandler(h http.Handler) http.Handler {
	return web.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return s.html.Render("RequestEntityTooLarge", nil)
		},
		ContentType:  "text/html; charset=utf-8",
		ErrorHandler: s.htmlServerError,
	}
}

func textMaxBodyBytesHandler(h http.Handler) http.Handler {
	return web.MaxBodyBytesHandler{
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
	return web.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return `{"message":"Request Entity Too Large","code":413}`, nil
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

func generateAntiXSRFCookieHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(antixsrf.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) jsonAntiXSRFHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := antixsrf.Verify(r); err != nil {
			s.Logger.Warningf("xsrf %s: %s", r.RequestURI, err)
			jsonresponse.Forbidden(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) htmlLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.Logger.Errorf("email notify: %s", err)
				}
			}()
			s.htmlServerError(w, r, err)
			return
		}
		if r.Header.Get(antixsrf.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		if u == nil || u.Disabled {
			if r.Header.Get(s.SessionCookieName) != "" {
				s.logout(w, r)
			}
			s.html.Respond(w, "Login", nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) jsonLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.Logger.Errorf("email notify: %s", err)
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
		u, r, err := s.getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.Logger.Errorf("email notify: %s", err)
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
		u, r, err := s.getRequestUser(r)
		if err != nil {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.Logger.Errorf("email notify: %s", err)
				}
			}()
			s.htmlServerError(w, r, err)
			return
		}
		if u.EmailUnvalidated {
			s.html.Respond(w, "EmailUnvalidated", map[string]interface{}{
				"User": u,
			})
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) jsonValidatedEmailRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := s.getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer s.RecoveryService.Recover()
				if err := s.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					s.Logger.Errorf("email notify: %s", err)
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
	web.HandleMethods(h, "Method Not Allowed", "text/plain; charset=utf-8", w, r)
}

type jsonMethodHandler map[string]http.Handler

func (h jsonMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	web.HandleMethods(h, `{"message":"Method Not Allowed","code":405}`, "application/json; charset=utf-8", w, r)
}

func noCacheHeaderHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

func (s *Server) apiDisabledHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.APIEnabled {
			jsonresponse.NotFound(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) apiKeyAuthHandler(h http.Handler, body, contentType string) http.Handler {
	trustedProxyNetworks := []net.IPNet{}
	for _, cidr := range s.APITrustedProxyCIDRs {
		if cidr == "" {
			continue
		}
		_, cidrnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		trustedProxyNetworks = append(trustedProxyNetworks, *cidrnet)
	}
	return web.AuthHandler{
		Handler: h,
		UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			jsonresponse.Unauthorized(w, nil)
		}),
		AuthFunc: func(r *http.Request, field1, field2 string) (valid bool, entity interface{}, err error) {
			if field1 == "" {
				field1 = field2
			}
			if field1 == "" {
				return
			}
			k, err := s.KeyService.KeyBySecret(field1)
			switch err {
			case nil:
			case key.ErrKeyNotFound:
				err = nil
				return
			default:
				err = nil
				s.Logger.Errorf("api key auth: get key: %s", err)
				return
			}

			var host string
			host, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)
			if ip == nil {
				s.Logger.Warningf("api key auth: key ref %s: unable to parse ip: %s", k.Ref, host)
				return
			}

			if len(trustedProxyNetworks) > 0 && s.APIProxyRealIPHeader != "" {
				proxied := false
				for _, network := range trustedProxyNetworks {
					if network.Contains(ip) {
						proxied = true
						break
					}
				}
				if proxied {
					header := r.Header.Get(s.APIProxyRealIPHeader)
					if header != "" {
						header = strings.TrimSpace(strings.SplitN(header, ",", 2)[0])
						ip = net.ParseIP(header)
						if ip == nil {
							s.Logger.Warningf("api key auth: key ref %s: unable to parse %s header as ip: %s", k.Ref, s.APIProxyRealIPHeader, header)
							return
						}
					}
				}
			}

			found := false
			for _, net := range k.AuthorizedNetworks {
				if net.Contains(ip) {
					found = true
					break
				}
			}
			if !found {
				s.Logger.Warningf("api key auth: key ref %s: unauthorized network: %s", k.Ref, ip)
				return
			}

			entity, err = s.UserService.UserByID(k.Ref)
			if err != nil {
				err = nil
				s.Logger.Errorf("api key auth: get user by id %s: %s", k.Ref, err)
				return
			}
			valid = true
			return
		},
		PostAuthFunc: func(_ http.ResponseWriter, r *http.Request, valid bool, entity interface{}) (rr *http.Request, err error) {
			if valid && entity != nil {
				rr = r.WithContext(context.WithValue(r.Context(), contextKeyUser, entity))
			}
			return
		},
		KeyHeaderName:  "X-Key",
		BasicAuthRealm: "Key",
	}
}

func (s *Server) jsonAPIKeyAuthHandler(h http.Handler) http.Handler {
	return s.apiKeyAuthHandler(h, `{"message":"Unauthorized","code":401}`, "application/json; charset=utf-8")
}

func (s *Server) redirectHandler(h http.Handler) (http.Handler, error) {
	if s.ListenTLS == "" || s.Domain == "" {
		return h, nil
	}
	// Initialize handler that will redirect http:// to https:// only if
	// certificate for configured domain or it's www subdomain is available.
	_, tlsPort, err := net.SplitHostPort(s.ListenTLS)
	if err != nil {
		return nil, fmt.Errorf("invalid tls: %s", err)
	}
	if tlsPort == "443" {
		tlsPort = ""
	} else {
		tlsPort = ":" + tlsPort
	}
	var altDomain string
	if strings.HasPrefix("www.", s.Domain) {
		altDomain = strings.TrimPrefix(s.Domain, "www.")
	} else {
		altDomain = "www." + s.Domain
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			domain = r.Host
		}
		if (domain == s.Domain || domain == altDomain) && !strings.HasPrefix(r.URL.Path, acmeURLPrefix) {
			c, _ := s.certificateCache.Certificate(s.Domain)
			if c != nil {
				http.Redirect(w, r, strings.Join([]string{"https://", s.Domain, tlsPort, r.RequestURI}, ""), http.StatusMovedPermanently)
				return
			}
		}
		h.ServeHTTP(w, r)
	}), nil
}
