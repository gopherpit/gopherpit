// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"resenje.org/httputils"
)

func setupFrontendRouter(baseRouter *http.ServeMux) {
	frontendRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/", chainHandlers(
		handlers.CompressHandler,
		htmlRecoveryHandler,
		accessLogHandler,
		htmlMaintenanceHandler,
		htmlMaxBodyBytesHandler,
		acmeUserHandler,
		finalHandler(frontendRouter),
	))
	// Frontend routes start
	frontendRouter.NotFoundHandler = chainHandlers(
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, map[string]string{
				"Cache-Control": "no-cache",
			})
		},
		func(h http.Handler) http.Handler {
			return httputils.NewStaticFilesHandler(h, "/", http.Dir(srv.StaticDir))
		},
		finalHandlerFunc(htmlNotFoundHandler),
	)
	frontendRouter.Handle("/", htmlLoginAltHandler(
		chainHandlers(
			htmlValidatedEmailRequiredHandler,
			finalHandlerFunc(dashboardHandler),
		),
		chainHandlers(
			generateAntiXSRFCookieHandler,
			finalHandlerFunc(landingPageHandler),
		),
	))
	frontendRouter.Handle("/about", http.HandlerFunc(aboutHandler))
	frontendRouter.Handle("/license", http.HandlerFunc(licenseHandler))
	frontendRouter.Handle("/contact", chainHandlers(
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(contactHandler),
	))
	frontendRouter.Handle("/login", chainHandlers(
		htmlLoginRequiredHandler,
		finalHandler(http.RedirectHandler("/", http.StatusSeeOther)),
	))
	frontendRouter.Handle("/logout", http.HandlerFunc(logoutHandler))
	frontendRouter.Handle("/registration", htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		chainHandlers(
			generateAntiXSRFCookieHandler,
			finalHandlerFunc(registrationHandler),
		),
	))
	frontendRouter.Handle("/password-reset", htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		chainHandlers(
			generateAntiXSRFCookieHandler,
			finalHandlerFunc(passwordResetTokenHandler),
		),
	))
	frontendRouter.Handle(`/password-reset/{token}`, chainHandlers(
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(passwordResetHandler),
	))
	frontendRouter.Handle(`/email/{token}`, chainHandlers(
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(publicEmailSettingsHandler),
	))
	frontendRouter.Handle(`/email-validation/{token}`, chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(emailValidationHandler),
	))
	frontendRouter.Handle("/settings", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(settingsHandler),
	))
	frontendRouter.Handle("/settings/email", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(settingsEmailHandler),
	))
	frontendRouter.Handle("/settings/notifications", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(settingsNotificationsHandler),
	))
	frontendRouter.Handle("/settings/password", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(settingsPasswordHandler),
	))
	frontendRouter.Handle("/settings/api", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(apiAccessSettingsHandler),
	))
	frontendRouter.Handle("/settings/delete-account", chainHandlers(
		htmlLoginRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(settingsDeleteAccountHandler),
	))

	frontendRouter.Handle("/domain", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainAddHandler),
	))
	frontendRouter.Handle("/domain/{id}", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		finalHandlerFunc(domainPackagesHandler),
	))
	frontendRouter.Handle("/domain/{id}/settings", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainSettingsHandler),
	))
	frontendRouter.Handle("/domain/{id}/team", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainTeamHandler),
	))
	frontendRouter.Handle("/domain/{id}/changelog", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainChangelogHandler),
	))
	frontendRouter.Handle("/domain/{id}/user", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainDomainUserGrantHandler),
	))
	frontendRouter.Handle("/domain/{id}/user/{user-id}/revoke", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainDomainUserRevokeHandler),
	))
	frontendRouter.Handle("/domain/{id}/owner", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainDomainOwnerChangeHandler),
	))
	frontendRouter.Handle("/domain/{domain-id}/package", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainPackageEditHandler),
	))
	frontendRouter.Handle("/package/{package-id}", chainHandlers(
		htmlLoginRequiredHandler,
		htmlValidatedEmailRequiredHandler,
		generateAntiXSRFCookieHandler,
		finalHandlerFunc(domainPackageEditHandler),
	))
	frontendRouter.Handle("/user/{id}", chainHandlers(
		htmlLoginRequiredHandler,
		finalHandlerFunc(userPageHandler),
	))
	// Frontend routes end
}
