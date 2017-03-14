// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func setupAPIRouter(baseRouter *http.ServeMux) {
	apiRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/api/", chainHandlers(
		handlers.CompressHandler,
		jsonRecoveryHandler,
		accessLogHandler,
		apiDisabledHandler,
		jsonMaintenanceHandler,
		jsonMaxBodyBytesHandler,
		jsonAPIKeyAuthHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(apiRouter),
	))
	apiRouter.NotFoundHandler = http.HandlerFunc(jsonNotFoundHandler)
	// API routes start
	apiRouter.Handle("/api/v1/domains", jsonMethodHandler{
		"GET": http.HandlerFunc(domainsAPIHandler),
		"POST": chainHandlers(
			jsonAPIRateLimiterHandler,
			finalHandlerFunc(updateDomainAPIHandler),
		),
	})
	apiRouter.Handle("/api/v1/domains/{id}", jsonMethodHandler{
		"GET": http.HandlerFunc(domainAPIHandler),
		"POST": chainHandlers(
			jsonAPIRateLimiterHandler,
			finalHandlerFunc(updateDomainAPIHandler),
		),
		"DELETE": http.HandlerFunc(deleteDomainAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/tokens", jsonMethodHandler{
		"GET": http.HandlerFunc(domainTokensAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/users", jsonMethodHandler{
		"GET": http.HandlerFunc(domainUsersAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/users/{user-id}", jsonMethodHandler{
		"POST":   http.HandlerFunc(grantDomainUserAPIHandler),
		"DELETE": http.HandlerFunc(revokeDomainUserAPIHandler),
	})
	apiRouter.Handle("/api/v1/domains/{id}/packages", jsonMethodHandler{
		"GET": http.HandlerFunc(domainPackagesAPIHandler),
	})
	apiRouter.Handle("/api/v1/packages", jsonMethodHandler{
		"POST": http.HandlerFunc(updatePackageAPIHandler),
	})
	apiRouter.Handle("/api/v1/packages/{id}", jsonMethodHandler{
		"GET":    http.HandlerFunc(packageAPIHandler),
		"POST":   http.HandlerFunc(updatePackageAPIHandler),
		"DELETE": http.HandlerFunc(deletePackageAPIHandler),
	})
	// API routes end
}
