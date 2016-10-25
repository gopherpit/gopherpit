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
	"time"

	"resenje.org/antixsrf"
	"resenje.org/logging"

	"gopherpit.com/gopherpit/pkg/webutils"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

const (
	acmeURLPrefix    = "/.well-known/acme-challenge/"
	acmeURLPrefixLen = 28
)

func (s Server) domainHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rDomain, port, err := net.SplitHostPort(r.Host)
		if err != nil {
			rDomain = r.Host
		}
		// Handle ACME challenges.
		if strings.HasPrefix(r.URL.Path, acmeURLPrefix) {
			s.logger.Debugf("domain: acme challenge: %s%s", r.Host, r.URL.String())
			token := r.URL.Path[acmeURLPrefixLen:]
			if token != "" {
				c, err := s.CertificateService.ACMEChallenge(rDomain)
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
				s.logger.Warningf("domain: acme challenge: %s: invalid token %s", r.URL.String(), c.Token)
			}
		}
		// Redirect www to naked domain
		if rDomain == "www."+s.Domain {
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
			http.Redirect(w, r, strings.Join([]string{scheme, "://", s.Domain, port, r.URL.Path, query}, ""), http.StatusMovedPermanently)
			return
		}
		// Handle packages.
		if s.Domain != "" && rDomain != s.Domain {
			s.packageResolverHandler(w, r)
			return
		}
		// Handle main site.
		h.ServeHTTP(w, r)
	})
}

func (s Server) acmeUserHandler(h http.Handler) http.Handler {
	registerUser := &s.TLSEnabled
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *registerUser {
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

			au, err := s.CertificateService.ACMEUser()
			if err != nil {
				if err == certificate.ACMEUserNotFound {
					if r.Header.Get(s.XSRFCookieName) == "" {
						antixsrf.Generate(w, r, "/")
					}
					if u != nil {
						respond(w, s.template(tidRegisterACMEUserPrivate), map[string]interface{}{
							"User":                u,
							"ProductionDirectory": s.ACMEDirectoryURL,
							"StagingDirectory":    s.ACMEDirectoryURLStaging,
						})
						return
					}
					respond(w, s.template(tidRegisterACMEUser), map[string]interface{}{
						"ProductionDirectory": s.ACMEDirectoryURL,
						"StagingDirectory":    s.ACMEDirectoryURLStaging,
					})
					return
				}
				s.logger.Errorf("acme user: get acme user: %s", err)
				s.htmlInternalServerErrorHandler(w, r)
				return
			}
			if au != nil {
				*registerUser = false
			}
		}
		h.ServeHTTP(w, r)
	})
}

func (s Server) packageResolverHandler(w http.ResponseWriter, r *http.Request) {
	var code int
	defer func(startTime time.Time) {
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
		case code >= 500:
			level = logging.ERROR
		case code >= 400:
			level = logging.WARNING
		case code >= 300:
			level = logging.INFO
		case code >= 200:
			level = logging.INFO
		default:
			level = logging.DEBUG
		}
		s.packageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, webutils.GetHost(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + r.URL.Path
	resolution, err := s.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.DomainNotFound || err == packages.PackageNotFound {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			code = http.StatusNotFound
			w.WriteHeader(code)
			fmt.Fprintln(w, fmt.Sprintf("No packages found for path %s", path))
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		code = http.StatusInternalServerError
		w.WriteHeader(code)
		fmt.Fprintln(w, http.StatusText(http.StatusInternalServerError))
		return
	}

	if resolution.Disabled {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		code = http.StatusNotFound
		w.WriteHeader(code)
		fmt.Fprintln(w, fmt.Sprintf("No packages found for path %s", path))
		return
	}

	code = 200
	respond(w, s.template(tidPackageResolution), map[string]interface{}{
		"GoImport":    fmt.Sprintf("%s %s %s", resolution.ImportPrefix, resolution.VCS, resolution.RepoRoot),
		"GoSource":    resolution.GoSource,
		"RedirectURL": resolution.RedirectURL,
	})
}
