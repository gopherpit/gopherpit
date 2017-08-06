// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"resenje.org/web"

	"github.com/gorilla/mux"
)

func newAPIRouter(s *Server) http.Handler {
	apiRouter := mux.NewRouter().StrictSlash(true)
	apiRouter.NotFoundHandler = http.HandlerFunc(jsonNotFoundHandler)
	// API routes start
	apiRouter.Handle("/api/v1/domains", jsonMethodHandler{
		"GET": http.HandlerFunc(s.domainsAPIHandler),
		"POST": web.ChainHandlers(
			s.jsonAPIRateLimiterHandler,
			web.FinalHandlerFunc(s.updateDomainAPIHandler),
		),
	})
	apiRouter.Handle("/api/v1/domains/{id}", jsonMethodHandler{
		"GET": http.HandlerFunc(s.domainAPIHandler),
		"POST": web.ChainHandlers(
			s.jsonAPIRateLimiterHandler,
			web.FinalHandlerFunc(s.updateDomainAPIHandler),
		),
		"DELETE": http.HandlerFunc(s.deleteDomainAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/tokens", jsonMethodHandler{
		"GET": http.HandlerFunc(s.domainTokensAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/users", jsonMethodHandler{
		"GET": http.HandlerFunc(s.domainUsersAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/users/{user-id}", jsonMethodHandler{
		"POST":   http.HandlerFunc(s.grantDomainUserAPIHandler),
		"DELETE": http.HandlerFunc(s.revokeDomainUserAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/packages", jsonMethodHandler{
		"GET": http.HandlerFunc(s.domainPackagesAPIHandler),
	})
	apiRouter.Handle("/api/v1/packages", jsonMethodHandler{
		"POST": http.HandlerFunc(s.updatePackageAPIHandler),
	})
	apiRouter.Handle("/api/v1/packages/{id}", jsonMethodHandler{
		"GET":    http.HandlerFunc(s.packageAPIHandler),
		"POST":   http.HandlerFunc(s.updatePackageAPIHandler),
		"DELETE": http.HandlerFunc(s.deletePackageAPIHandler),
	})
	// API routes end
	return apiRouter
}
