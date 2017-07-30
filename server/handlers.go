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

	"gopherpit.com/gopherpit/pkg/info"
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

// Helper function for raising unexpected errors in HTML frontend handlers.
func htmlServerError(w http.ResponseWriter, r *http.Request, err error) {
	if _, ok := err.(net.Error); ok {
		htmlServiceUnavailableHandler(w, r)
		return
	}
	htmlInternalServerErrorHandler(w, r)
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

func statusAPIHandler(w http.ResponseWriter, r *http.Request) {
	jsonresponse.OK(w, statusResponse{
		Name:        srv.Name,
		Version:     version(),
		Uptime:      time.Since(srv.startTime).String(),
		Information: *info.NewInformation(),
	})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s version %s, uptime %s", srv.Name, version(), time.Since(srv.startTime))
}

func accessLogHandler(h http.Handler) http.Handler {
	return accessLog.NewHandler(h, srv.AccessLogger)
}

// Recovery handler for HTML frontend routers.
func htmlRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(version()),
		recovery.WithLogFunc(srv.Logger.Errorf),
		recovery.WithNotifier(srv.EmailService),
		recovery.WithPanicResponseHandler(http.HandlerFunc(htmlInternalServerErrorHandler)),
	)
}

// Recovery handler for JSON API routers.
func jsonRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(version()),
		recovery.WithLogFunc(srv.Logger.Errorf),
		recovery.WithNotifier(srv.EmailService),
		recovery.WithPanicResponse(`{"message":"Internal Server Error","code":500}`, "application/json; charset=utf-8"),
	)
}

// Recovery handler that does not write anything to response.
// It is useful as the firs handler in chain if a handler that
// transforms response needs to be before other recovery handler,
// like compress handler.
func nilRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(version()),
		recovery.WithLogFunc(srv.Logger.Errorf),
		recovery.WithNotifier(srv.EmailService),
	)
}

func htmlMaxBodyBytesHandler(h http.Handler) http.Handler {
	return web.MaxBodyBytesHandler{
		Handler: h,
		Limit:   2 * 1024 * 1024,
		BodyFunc: func(r *http.Request) (string, error) {
			return renderToString(srv.templates["RequestEntityTooLarge"], "", nil)
		},
		ContentType:  "text/html; charset=utf-8",
		ErrorHandler: htmlServerError,
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
		if r.Header.Get(srv.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		h.ServeHTTP(w, r)
	})
}

func jsonAntiXSRFHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := antixsrf.Verify(r); err != nil {
			srv.Logger.Warningf("xsrf %s: %s", r.RequestURI, err)
			jsonresponse.Forbidden(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func htmlLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer srv.RecoveryService.Recover()
				if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					srv.Logger.Errorf("email notify: %s", err)
				}
			}()
			htmlServerError(w, r, err)
			return
		}
		if r.Header.Get(srv.XSRFCookieName) == "" {
			antixsrf.Generate(w, r, "/")
		}
		if u == nil || u.Disabled {
			if r.Header.Get(srv.SessionCookieName) != "" {
				logout(w, r)
			}
			respond(w, "Login", nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func jsonLoginRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer srv.RecoveryService.Recover()
				if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					srv.Logger.Errorf("email notify: %s", err)
				}
			}()
			jsonServerError(w, err)
			return
		}
		if u == nil || u.Disabled {
			if r.Header.Get(srv.SessionCookieName) != "" {
				logout(w, r)
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
func htmlLoginAltHandler(h, alt http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer srv.RecoveryService.Recover()
				if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					srv.Logger.Errorf("email notify: %s", err)
				}
			}()
			htmlServerError(w, r, err)
			return
		}
		if u == nil || u.Disabled {
			if r.Header.Get(srv.SessionCookieName) != "" {
				logout(w, r)
			}
			alt.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func htmlValidatedEmailRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := getRequestUser(r)
		if err != nil {
			go func() {
				defer srv.RecoveryService.Recover()
				if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					srv.Logger.Errorf("email notify: %s", err)
				}
			}()
			htmlServerError(w, r, err)
			return
		}
		if u.EmailUnvalidated {
			respond(w, "EmailUnvalidated", map[string]interface{}{
				"User": u,
			})
			return
		}
		h.ServeHTTP(w, r)
	})
}

func jsonValidatedEmailRequiredHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, r, err := getRequestUser(r)
		if err != nil && err != user.ErrUserNotFound {
			go func() {
				defer srv.RecoveryService.Recover()
				if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
					srv.Logger.Errorf("email notify: %s", err)
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

func apiDisabledHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !srv.APIEnabled {
			jsonresponse.NotFound(w, nil)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func apiKeyAuthHandler(h http.Handler, body, contentType string) http.Handler {
	trustedProxyNetworks := []net.IPNet{}
	for _, cidr := range srv.APITrustedProxyCIDRs {
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
			k, err := srv.KeyService.KeyBySecret(field1)
			switch err {
			case nil:
			case key.ErrKeyNotFound:
				err = nil
				return
			default:
				err = nil
				srv.Logger.Errorf("api key auth: get key: %s", err)
				return
			}

			var host string
			host, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip := net.ParseIP(host)
			if ip == nil {
				srv.Logger.Warningf("api key auth: key ref %s: unable to parse ip: %s", k.Ref, host)
				return
			}

			if len(trustedProxyNetworks) > 0 && srv.APIProxyRealIPHeader != "" {
				proxied := false
				for _, network := range trustedProxyNetworks {
					if network.Contains(ip) {
						proxied = true
						break
					}
				}
				if proxied {
					header := r.Header.Get(srv.APIProxyRealIPHeader)
					if header != "" {
						header = strings.TrimSpace(strings.SplitN(header, ",", 2)[0])
						ip = net.ParseIP(header)
						if ip == nil {
							srv.Logger.Warningf("api key auth: key ref %s: unable to parse %s header as ip: %s", k.Ref, srv.APIProxyRealIPHeader, header)
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
				srv.Logger.Warningf("api key auth: key ref %s: unauthorized network: %s", k.Ref, ip)
				return
			}

			entity, err = srv.UserService.UserByID(k.Ref)
			if err != nil {
				err = nil
				srv.Logger.Errorf("api key auth: get user by id %s: %s", k.Ref, err)
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

func jsonAPIKeyAuthHandler(h http.Handler) http.Handler {
	return apiKeyAuthHandler(h, `{"message":"Unauthorized","code":401}`, "application/json; charset=utf-8")
}
