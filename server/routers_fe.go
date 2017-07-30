// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"resenje.org/web"
)

func setupFrontendRouter(baseRouter *http.ServeMux) {
	frontendRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		htmlRecoveryHandler,
		accessLogHandler,
		htmlMaintenanceHandler,
		htmlMaxBodyBytesHandler,
		acmeUserHandler,
		web.FinalHandler(frontendRouter),
	))
	// Frontend routes start
	frontendRouter.NotFoundHandler = web.ChainHandlers(
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, map[string]string{
				"Cache-Control": "no-cache",
			})
		},
		func(h http.Handler) http.Handler {
			return web.NewStaticFilesHandler(h, "/", http.Dir(srv.StaticDir))
		},
		web.FinalHandlerFunc(htmlNotFoundHandler),
	)
	frontendRouter.Handle("/", htmlLoginAltHandler(
		web.ChainHandlers(
			htmlValidatedEmailRequiredHandler,
			web.FinalHandlerFunc(dashboardHandler),
		),
		web.ChainHandlers(
			generateAntiXSRFCookieHandler,
			web.FinalHandlerFunc(landingPageHandler),
		),
	))
	frontendRouter.Handle("/about", http.HandlerFunc(aboutHandler))
	frontendRouter.Handle("/license", http.HandlerFunc(licenseHandler))
	frontendRouter.Handle("/docs/api", http.HandlerFunc(apiDocsHandler))
	frontendRouter.Handle("/contact", web.ChainHandlers(
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(contactHandler),
	))
	frontendRouter.Handle("/login", web.ChainHandlers(
		htmlLoginRequiredHandler,
		web.FinalHandler(http.RedirectHandler("/", http.StatusSeeOther)),
	))
	frontendRouter.Handle("/logout", http.HandlerFunc(logoutHandler))
	frontendRouter.Handle("/registration", htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		web.ChainHandlers(
			generateAntiXSRFCookieHandler,
			web.FinalHandlerFunc(registrationHandler),
		),
	))
	frontendRouter.Handle("/password-reset", htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		web.ChainHandlers(
			generateAntiXSRFCookieHandler,
			web.FinalHandlerFunc(passwordResetTokenHandler),
		),
	))
	frontendRouter.Handle(`/password-reset/{token}`, web.ChainHandlers(
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(passwordResetHandler),
	))
	frontendRouter.Handle(`/email/{token}`, web.ChainHandlers(
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(publicEmailSettingsHandler),
	))
	frontendRouter.Handle(`/email-validation/{token}`, web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(emailValidationHandler),
	))
	frontendRouter.Handle("/settings", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(settingsHandler),
	))
	frontendRouter.Handle("/settings/email", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(settingsEmailHandler),
	))
	frontendRouter.Handle("/settings/notifications", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(settingsNotificationsHandler),
	))
	frontendRouter.Handle("/settings/password", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(settingsPasswordHandler),
	))
	frontendRouter.Handle("/settings/api", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(apiAccessSettingsHandler),
	))
	frontendRouter.Handle("/settings/delete-account", web.ChainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(settingsDeleteAccountHandler),
	))

	frontendRouter.Handle("/domain", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainAddHandler),
	))
	frontendRouter.Handle("/domain/{id}", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		web.FinalHandlerFunc(domainPackagesHandler),
	))
	frontendRouter.Handle("/domain/{id}/settings", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainSettingsHandler),
	))
	frontendRouter.Handle("/domain/{id}/team", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainTeamHandler),
	))
	frontendRouter.Handle("/domain/{id}/changelog", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainChangelogHandler),
	))
	frontendRouter.Handle("/domain/{id}/user", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainDomainUserGrantHandler),
	))
	frontendRouter.Handle("/domain/{id}/user/{user-id}/revoke", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainDomainUserRevokeHandler),
	))
	frontendRouter.Handle("/domain/{id}/owner", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainDomainOwnerChangeHandler),
	))
	frontendRouter.Handle("/domain/{domain-id}/package", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainPackageEditHandler),
	))
	frontendRouter.Handle("/package/{package-id}", web.ChainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		web.FinalHandlerFunc(domainPackageEditHandler),
	))
	frontendRouter.Handle("/user/{id}", web.ChainHandlers(
		htmlLoginRequiredHandler,
		web.FinalHandlerFunc(userPageHandler),
	))
	// Frontend routes end
}
