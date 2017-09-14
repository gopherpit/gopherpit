// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"
	"path/filepath"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"resenje.org/antixsrf"
	"resenje.org/web"

	"gopherpit.com/gopherpit/server/data/static"
)

func newFrontendRouter(s *Server) http.Handler {
	frontendRouter := mux.NewRouter().StrictSlash(true)
	// Frontend routes start
	frontendRouter.NotFoundHandler = web.ChainHandlers(
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, map[string]string{
				"Cache-Control": "no-cache",
			})
		},
		func(h http.Handler) http.Handler {
			return web.NewStaticFilesHandler(h, "/", http.Dir(filepath.Join(s.StorageDir, "static")))
		},
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fs := &assetfs.AssetFS{Asset: static.Asset, AssetDir: static.AssetDir, AssetInfo: static.AssetInfo}
				web.NewStaticFilesHandler(h, "/", fs).ServeHTTP(w, r)
			})
		},
		web.FinalHandlerFunc(s.htmlNotFoundHandler),
	)
	frontendRouter.Handle("/", s.htmlLoginAltHandler(
		web.ChainHandlers(
			s.htmlValidatedEmailRequiredHandler,
			web.FinalHandlerFunc(s.dashboardHandler),
		),
		web.ChainHandlers(
			antixsrf.GenerateHandler,
			web.FinalHandlerFunc(s.landingPageHandler),
		),
	))
	frontendRouter.Handle("/about", http.HandlerFunc(s.aboutHandler))
	frontendRouter.Handle("/license", http.HandlerFunc(s.licenseHandler))
	frontendRouter.Handle("/docs/api", http.HandlerFunc(s.apiDocsHandler))
	frontendRouter.Handle("/contact", web.ChainHandlers(
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.contactHandler),
	))
	frontendRouter.Handle("/login", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		web.FinalHandler(http.RedirectHandler("/", http.StatusSeeOther)),
	))
	frontendRouter.Handle("/logout", http.HandlerFunc(s.logoutHandler))
	frontendRouter.Handle("/registration", s.htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		web.ChainHandlers(
			antixsrf.GenerateHandler,
			web.FinalHandlerFunc(s.registrationHandler),
		),
	))
	frontendRouter.Handle("/password-reset", s.htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		web.ChainHandlers(
			antixsrf.GenerateHandler,
			web.FinalHandlerFunc(s.passwordResetTokenHandler),
		),
	))
	frontendRouter.Handle(`/password-reset/{token}`, web.ChainHandlers(
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.passwordResetHandler),
	))
	frontendRouter.Handle(`/email/{token}`, web.ChainHandlers(
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.publicEmailSettingsHandler),
	))
	frontendRouter.Handle(`/email-validation/{token}`, web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.emailValidationHandler),
	))
	frontendRouter.Handle("/settings", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.settingsHandler),
	))
	frontendRouter.Handle("/settings/email", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.settingsEmailHandler),
	))
	frontendRouter.Handle("/settings/notifications", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.settingsNotificationsHandler),
	))
	frontendRouter.Handle("/settings/password", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.settingsPasswordHandler),
	))
	frontendRouter.Handle("/settings/api", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.apiAccessSettingsHandler),
	))
	frontendRouter.Handle("/settings/delete-account", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.settingsDeleteAccountHandler),
	))

	frontendRouter.Handle("/domain", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainAddHandler),
	))
	frontendRouter.Handle("/domain/{id}", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		web.FinalHandlerFunc(s.domainPackagesHandler),
	))
	frontendRouter.Handle("/domain/{id}/settings", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainSettingsHandler),
	))
	frontendRouter.Handle("/domain/{id}/team", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainTeamHandler),
	))
	frontendRouter.Handle("/domain/{id}/changelog", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainChangelogHandler),
	))
	frontendRouter.Handle("/domain/{id}/user", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainDomainUserGrantHandler),
	))
	frontendRouter.Handle("/domain/{id}/user/{user-id}/revoke", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainDomainUserRevokeHandler),
	))
	frontendRouter.Handle("/domain/{id}/owner", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainDomainOwnerChangeHandler),
	))
	frontendRouter.Handle("/domain/{domain-id}/package", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainPackageEditHandler),
	))
	frontendRouter.Handle("/package/{package-id}", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		antixsrf.GenerateHandler,
		web.FinalHandlerFunc(s.domainPackageEditHandler),
	))
	frontendRouter.Handle("/user/{id}", web.ChainHandlers(
		s.htmlLoginRequiredHandler,
		web.FinalHandlerFunc(s.userPageHandler),
	))
	// Frontend routes end
	return frontendRouter
}
