// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"expvar"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/handlers"
	"resenje.org/web"
	"resenje.org/web/file-server"
	"resenje.org/x/data-dump"
)

func newRouter(s *Server, assetsServer *fileServer.Server) http.Handler {
	baseRouter := http.NewServeMux()

	baseRouter.Handle("/assets/", web.ChainHandlers(
		handlers.CompressHandler,
		s.htmlRecoveryHandler,
		s.accessLogHandler,
		s.htmlMaxBodyBytesHandler,
		web.NoExpireHeadersHandler,
		web.FinalHandler(assetsServer),
	))
	baseRouter.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		s.htmlRecoveryHandler,
		s.accessLogHandler,
		s.MaintenanceService.HTMLHandler,
		s.htmlMaxBodyBytesHandler,
		s.acmeUserHandler,
		web.FinalHandler(newFrontendRouter(s)),
	))
	baseRouter.Handle("/i/", web.ChainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		s.accessLogHandler,
		s.MaintenanceService.JSONHandler,
		s.jsonAntiXSRFHandler,
		jsonMaxBodyBytesHandler,
		web.FinalHandler(newFrontendAPIRouter(s)),
	))
	baseRouter.Handle("/api/", web.ChainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		s.accessLogHandler,
		s.apiDisabledHandler,
		s.MaintenanceService.JSONHandler,
		jsonMaxBodyBytesHandler,
		s.jsonAPIKeyAuthHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(newAPIRouter(s)),
	))

	// Final handler
	return web.ChainHandlers(
		s.domainHandler,
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, s.Headers)
		},
		web.FinalHandler(baseRouter),
	)
}

func newInternalRouter(s *Server) http.Handler {
	internalBaseRouter := http.NewServeMux()

	//
	// Internal frontend router
	//
	internalRouter := http.NewServeMux()
	internalBaseRouter.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(internalRouter),
	))
	internalRouter.Handle("/", http.HandlerFunc(textNotFoundHandler))
	internalRouter.Handle("/status", http.HandlerFunc(s.statusHandler))
	internalRouter.Handle("/data", dataDump.Handler(s.options, s.Name+"_"+s.version(), s.Logger))

	internalRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	internalRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	internalRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	internalRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	internalRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	internalRouter.Handle("/debug/vars", expvar.Handler())

	//
	// Internal API router
	//
	internalAPIRouter := http.NewServeMux()
	internalBaseRouter.Handle("/api/", web.ChainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(internalAPIRouter),
	))
	internalAPIRouter.Handle("/api/", http.HandlerFunc(jsonNotFoundHandler))
	internalAPIRouter.Handle("/api/status", http.HandlerFunc(s.statusAPIHandler))
	internalAPIRouter.Handle("/api/maintenance", jsonMethodHandler{
		"GET":    http.HandlerFunc(s.MaintenanceService.StatusHandler),
		"POST":   http.HandlerFunc(s.MaintenanceService.OnHandler),
		"DELETE": http.HandlerFunc(s.MaintenanceService.OffHandler),
	})

	// Final internal handler
	return web.ChainHandlers(
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, s.Headers)
		},
		web.FinalHandler(internalBaseRouter),
	)
}
