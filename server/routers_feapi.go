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

func setupFrontendAPIRouter(baseRouter *http.ServeMux) {
	frontendAPIRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/i/", chainHandlers(
		handlers.CompressHandler,
		jsonRecoveryHandler,
		accessLogHandler,
		jsonMaintenanceHandler,
		jsonAntiXSRFHandler,
		jsonMaxBodyBytesHandler,
		finalHandler(frontendAPIRouter),
	))
	frontendAPIRouter.NotFoundHandler = http.HandlerFunc(jsonNotFoundHandler)
	// Frontend API routes start
	// ACME
	frontendAPIRouter.Handle("/i/register-acme-user", jsonMethodHandler{
		"POST": http.HandlerFunc(registerACMEUserFEAPIHandler),
	})
	// User public
	frontendAPIRouter.Handle("/i/auth", jsonMethodHandler{
		"POST":   http.HandlerFunc(authLoginFEAPIHandler),
		"DELETE": http.HandlerFunc(authLogoutFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/registration", jsonMethodHandler{
		"POST": http.HandlerFunc(registrationFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset-token", jsonMethodHandler{
		"POST": http.HandlerFunc(passwordResetTokenFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset", jsonMethodHandler{
		"POST": http.HandlerFunc(passwordResetFEAPIHandler),
	})
	frontendAPIRouter.Handle(`/i/email/opt-out/{token:\w{27,}}`, jsonMethodHandler{
		"POST":   http.HandlerFunc(emailOptOutFEAPIHandler),
		"DELETE": http.HandlerFunc(emailRemoveOptOutFEAPIHandler),
	})
	// Contact
	frontendAPIRouter.Handle("/i/contact", jsonMethodHandler{
		"POST": htmlLoginAltHandler(
			http.HandlerFunc(contactPrivateFEAPIHandler),
			http.HandlerFunc(contactFEAPIHandler),
		),
	})
	// User settings
	frontendAPIRouter.Handle("/i/user", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/email", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userEmailFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/notifications", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userNotificationsSettingsFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/email/validation-email", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userSendEmailValidationEmailFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/password", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userPasswordFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle("/i/user/delete", chainHandlers(
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(userDeleteFEAPIHandler),
		}),
	))
	// API settings
	frontendAPIRouter.Handle(`/i/api/key`, chainHandlers(
		apiDisabledHandler,
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(apiKeyFEAPIHandler),
			"DELETE": http.HandlerFunc(apiKeyDeleteFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/api/networks`, chainHandlers(
		apiDisabledHandler,
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(apiNetworksFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/api/secret`, chainHandlers(
		apiDisabledHandler,
		jsonLoginRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(apiRegenerateSecretFEAPIHandler),
		}),
	))
	// SSL Certificate
	frontendAPIRouter.Handle(`/i/certificate/{id}`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(certificateFEAPIHandler),
		}),
	))
	// Domain
	frontendAPIRouter.Handle(`/i/domain`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(domainFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(domainFEAPIHandler),
			"DELETE": http.HandlerFunc(domainDeleteFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}/user`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(domainUserGrantFEAPIHandler),
			"DELETE": http.HandlerFunc(domainUserRevokeFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/domain/{id}/owner`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(domainOwnerChangeFEAPIHandler),
		}),
	))
	// Package
	frontendAPIRouter.Handle(`/i/package`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST": http.HandlerFunc(packageFEAPIHandler),
		}),
	))
	frontendAPIRouter.Handle(`/i/package/{id}`, chainHandlers(
		jsonLoginRequiredHandler,
		jsonValidatedEmailRequiredHandler,
		finalHandler(jsonMethodHandler{
			"POST":   http.HandlerFunc(packageFEAPIHandler),
			"DELETE": http.HandlerFunc(packageDeleteFEAPIHandler),
		}),
	))
	// Frontend API routes end
}
