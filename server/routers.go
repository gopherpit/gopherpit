// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/handlers"
	"resenje.org/httputils"
)

func setupRouters() {
	if srv == nil {
		return
	}

	baseRouter := http.NewServeMux()

	baseRouter.Handle("/assets/", chainHandlers(
		handlers.CompressHandler,
		htmlRecoveryHandler,
		accessLogHandler,
		htmlMaxBodyBytesHandler,
		httputils.NoExpireHeadersHandler,
		finalHandler(srv.assetsServer),
	))
	setupFrontendRouter(baseRouter)
	setupFrontendAPIRouter(baseRouter)
	setupAPIRouter(baseRouter)

	// Final handler
	srv.handler = chainHandlers(
		domainHandler,
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, srv.Headers)
		},
		finalHandler(baseRouter),
	)
}

func setupInternalRouters() {
	if srv == nil {
		return
	}

	internalBaseRouter := http.NewServeMux()

	//
	// Internal frontend router
	//
	internalRouter := http.NewServeMux()
	internalBaseRouter.Handle("/", chainHandlers(
		handlers.CompressHandler,
		httputils.NoCacheHeadersHandler,
		finalHandler(internalRouter),
	))
	internalRouter.Handle("/", http.HandlerFunc(textNotFoundHandler))
	internalRouter.Handle("/status", http.HandlerFunc(statusHandler))
	internalRouter.Handle("/data", http.HandlerFunc(dataDumpHandler))

	internalRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	internalRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	internalRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	internalRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	internalRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	//
	// Internal API router
	//
	internalAPIRouter := http.NewServeMux()
	internalBaseRouter.Handle("/api/", chainHandlers(
		handlers.CompressHandler,
		jsonRecoveryHandler,
		httputils.NoCacheHeadersHandler,
		finalHandler(internalAPIRouter),
	))
	internalAPIRouter.Handle("/api/", http.HandlerFunc(jsonNotFoundHandler))
	internalAPIRouter.Handle("/api/status", http.HandlerFunc(statusAPIHandler))
	internalAPIRouter.Handle("/api/maintenance", jsonMethodHandler{
		"GET":    http.HandlerFunc(maintenanceStatusAPIHandler),
		"POST":   http.HandlerFunc(maintenanceOnAPIHandler),
		"DELETE": http.HandlerFunc(maintenanceOffAPIHandler),
	})

	// Final internal handler
	srv.internalHandler = chainHandlers(
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, srv.Headers)
		},
		finalHandler(internalBaseRouter),
	)
}
