// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net/http"
	"strconv"

	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

func packagesDomainToAPIDomain(d packages.Domain) api.Domain {
	return api.Domain{
		ID:                d.ID,
		FQDN:              d.FQDN,
		OwnerUserID:       d.OwnerUserID,
		CertificateIgnore: d.CertificateIgnore,
		Disabled:          d.Disabled,
	}
}

func packagesPackageToAPIPackage(p packages.Package, d *packages.Domain) api.Package {
	if d == nil {
		d = p.Domain
	}
	return api.Package{
		ID:          p.ID,
		DomainID:    d.ID,
		FQDN:        d.FQDN,
		Path:        p.Path,
		VCS:         api.VCS(p.VCS),
		RepoRoot:    p.RepoRoot,
		RefType:     api.RefType(p.RefType),
		RefName:     p.RefName,
		GoSource:    p.GoSource,
		RedirectURL: p.RedirectURL,
		Disabled:    p.Disabled,
	}
}

func (s *Server) jsonAPIRateLimiterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.APIHourlyRateLimit > 0 {
			var u *user.User
			var err error
			u, r, err = s.getRequestUser(r)
			if err != nil {
				panic(err)
			}
			limited, result, err := s.apiRateLimiter.RateLimit(fmt.Sprintf("userID:%s", u.ID), 1)
			if err != nil {
				s.Logger.Errorf("api rate limiter: rate limit: %s", err)
				jsonresponse.InternalServerError(w, nil)
				return
			}
			if result.Limit > 0 {
				w.Header().Set("X-Ratelimit-Limit", strconv.Itoa(result.Limit))
				w.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(result.Remaining))
				if result.ResetAfter > 0 {
					w.Header().Set("X-Ratelimit-Reset", fmt.Sprintf("%f", result.ResetAfter.Seconds()))
				}
				if result.RetryAfter > 0 {
					w.Header().Set("X-Ratelimit-Retry", fmt.Sprintf("%f", result.RetryAfter.Seconds()))
				}
			}
			if limited {
				s.Logger.Warningf("api rate limiter: blocked %s: retry after %s", u.ID, result.RetryAfter)
				jsonresponse.BadRequest(w, api.ErrTooManyRequests)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
