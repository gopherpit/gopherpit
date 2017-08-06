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

func newFrontendAPIRouter(s *Server) http.Handler {
	frontendAPIRouter := mux.NewRouter().StrictSlash(true)
	frontendAPIRouter.NotFoundHandler = http.HandlerFunc(jsonNotFoundHandler)
	// Frontend API routes start
	// ACME
	frontendAPIRouter.Handle("/i/register-acme-user", jsonMethodHandler{
		"POST": http.HandlerFunc(s.registerACMEUserFEAPIHandler),
	})
	// User public
	frontendAPIRouter.Handle("/i/auth", jsonMethodHandler{
		"POST":   http.HandlerFunc(s.authLoginFEAPIHandler),
		"DELETE": http.HandlerFunc(s.authLogoutFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/registration", jsonMethodHandler{
		"POST": http.HandlerFunc(s.registrationFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset-token", jsonMethodHandler{
		"POST": http.HandlerFunc(s.passwordResetTokenFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset", jsonMethodHandler{
		"POST": http.HandlerFunc(s.passwordResetFEAPIHandler),
	})
	frontendAPIRouter.Handle(`/i/email/opt-out/{token:\w{27,}}`, jsonMethodHandler{
		"POST":   http.HandlerFunc(s.emailOptOutFEAPIHandler),
		"DELETE": http.HandlerFunc(s.emailRemoveOptOutFEAPIHandler),
	})
	// Contact
	frontendAPIRouter.Handle("/i/contact", jsonMethodHandler{
		"POST": s.htmlLoginAltHandler(
			http.HandlerFunc(s.contactPrivateFEAPIHandler),
			http.HandlerFunc(s.contactFEAPIHandler),
		),
	})
	// User settings
	frontendAPIRouter.Handle("/i/user", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/email", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userEmailFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/notifications", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userNotificationsSettingsFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/email/validation-email", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userSendEmailValidationEmailFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/password", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userPasswordFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/delete", web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.userDeleteFEAPIHandler),
		}),
	))
	// API settings
	frontendAPIRouter.Handle(`/i/api/key`, web.ChainHandlers(
		s.apiDisabledHandler,
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(s.apiKeyFEAPIHandler),
			"DELETE": http.HandlerFunc(s.apiKeyDeleteFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/api/networks`, web.ChainHandlers(
		s.apiDisabledHandler,
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.apiNetworksFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/api/secret`, web.ChainHandlers(
		s.apiDisabledHandler,
		s.jsonLoginRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.apiRegenerateSecretFEAPIHandler),
		}),
	))
	// SSL Certificate
	frontendAPIRouter.Handle(`/i/certificate/{id}`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.certificateFEAPIHandler),
		}),
	))
	// Domain
	frontendAPIRouter.Handle(`/i/domain`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.domainFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(s.domainFEAPIHandler),
			"DELETE": http.HandlerFunc(s.domainDeleteFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}/user`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(s.domainUserGrantFEAPIHandler),
			"DELETE": http.HandlerFunc(s.domainUserRevokeFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}/owner`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.domainOwnerChangeFEAPIHandler),
		}),
	))
	// Package
	frontendAPIRouter.Handle(`/i/package`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(s.packageFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/package/{id}`, web.ChainHandlers(
		s.jsonLoginRequiredHandler,
		s.jsonValidatedEmailRequiredHandler,
		web.FinalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(s.packageFEAPIHandler),
			"DELETE": http.HandlerFunc(s.packageDeleteFEAPIHandler),
		}),
	))
	// Frontend API routes end
	return frontendAPIRouter
}
