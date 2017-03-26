// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"resenje.org/antixsrf"
	"resenje.org/logging"

	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/user"
)

const (
	acmeURLPrefix    = "/.well-known/acme-challenge/"
	acmeURLPrefixLen = 28
)

func domainHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rDomain, port, err := net.SplitHostPort(r.Host)
		if err != nil {
			rDomain = r.Host
		}
		// Handle ACME challenges.
		if strings.HasPrefix(r.URL.Path, acmeURLPrefix) {
			srv.Logger.Debugf("domain: acme challenge: %s%s", r.Host, r.URL.String())
			token := r.URL.Path[acmeURLPrefixLen:]
			if token != "" {
				c, err := srv.CertificateService.ACMEChallenge(rDomain)
				if err != nil {
					logging.Errorf("domain: %s: acme challenge: %s", rDomain, err)
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
					return
				}
				if c.Token == token {
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, c.KeyAuth)
					return
				}
				srv.Logger.Warningf("domain: acme challenge: %s: invalid token %s", r.URL.String(), c.Token)
			}
		}
		// Redirect www to naked domain
		if rDomain == "www."+srv.Domain {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			if port != "" {
				port = ":" + port
			}
			query := r.URL.RawQuery
			if query != "" {
				query = "?" + query
			}
			http.Redirect(w, r, strings.Join([]string{scheme, "://", srv.Domain, port, r.URL.Path, query}, ""), http.StatusMovedPermanently)
			return
		}
		// Handle packages.
		if srv.Domain != "" && rDomain != srv.Domain {
			packageResolverHandler(w, r)
			return
		}
		// Handle main site.
		h.ServeHTTP(w, r)
	})
}

func acmeUserHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if srv.registerACMEUser {
			u, r, err := getRequestUser(r)
			if err != nil && err != user.UserNotFound {
				go func() {
					defer srv.RecoveryService.Recover()
					if err := srv.EmailService.Notify("Get user error", fmt.Sprint(err)); err != nil {
						srv.Logger.Errorf("email notify: %s", err)
					}
				}()
				htmlServerError(w, r, err)
				return
			}

			au, err := srv.CertificateService.ACMEUser()
			if err != nil {
				if err == certificate.ACMEUserNotFound {
					if r.Header.Get(srv.XSRFCookieName) == "" {
						antixsrf.Generate(w, r, "/")
					}
					if u != nil {
						respond(w, "RegisterACMEUserPrivate", map[string]interface{}{
							"User":                u,
							"ProductionDirectory": srv.ACMEDirectoryURL,
							"StagingDirectory":    srv.ACMEDirectoryURLStaging,
						})
						return
					}
					respond(w, "RegisterACMEUser", map[string]interface{}{
						"ProductionDirectory": srv.ACMEDirectoryURL,
						"StagingDirectory":    srv.ACMEDirectoryURLStaging,
					})
					return
				}
				srv.Logger.Errorf("acme user: get acme user: %s", err)
				htmlInternalServerErrorHandler(w, r)
				return
			}
			if au != nil {
				srv.registerACMEUser = false
			}
		}
		h.ServeHTTP(w, r)
	})
}
