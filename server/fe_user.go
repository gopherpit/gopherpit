// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"

	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/user"
)

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	r, err := logout(w, r)
	if err != nil {
		srv.logger.Errorf("logout: %s", err)
		htmlServerError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, "Registration", nil)
}

func passwordResetHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, "PasswordReset", map[string]interface{}{
		"Token": mux.Vars(r)["token"],
	})
}

func passwordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, "PasswordResetToken", nil)
}

func emailValidationHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	vars := mux.Vars(r)
	token := vars["token"]

	u2, err := srv.UserService.ChangeEmail(u.ID, token)
	if err != nil {
		if terr, ok := err.(*user.Error); ok {
			srv.logger.Warningf("email validation: user %s: change email token %s: %s", u.ID, token, terr)
			respond(w, "EmailValidation", map[string]interface{}{
				"Valid": false,
				"User":  u,
			})
			return
		}
		srv.logger.Errorf("email validation: user %s: change email token %s: %s", u.ID, token, err)
		htmlServerError(w, r, err)
		return
	}
	respond(w, "EmailValidation", map[string]interface{}{
		"Valid": !u2.EmailUnvalidated,
		"User":  u2,
	})
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	respond(w, "Settings", map[string]interface{}{
		"User":       u,
		"APIEnabled": srv.APIEnabled,
	})
}

func settingsEmailHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := srv.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		srv.logger.Errorf("settings email: %s: is email opted out api: %s", u.Email, err)
		htmlServerError(w, r, err)
		return
	}
	respond(w, "SettingsEmail", map[string]interface{}{
		"User":       u,
		"OptedOut":   optedOut,
		"APIEnabled": srv.APIEnabled,
	})
}

func settingsNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := srv.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		srv.logger.Errorf("settings notifications: %s: is email opted out api: %s", u.Email, err)
		htmlServerError(w, r, err)
		return
	}
	respond(w, "SettingsNotifications", map[string]interface{}{
		"User":       u,
		"OptedOut":   optedOut,
		"APIEnabled": srv.APIEnabled,
	})
}

func settingsPasswordHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	respond(w, "SettingsPassword", map[string]interface{}{
		"User":       u,
		"APIEnabled": srv.APIEnabled,
	})
}

func apiAccessSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.APIEnabled {
		htmlNotFoundHandler(w, r)
		return
	}
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	k, err := srv.KeyService.KeyByRef(u.ID)
	switch err {
	case nil:
	case key.KeyNotFound:
		k = &key.Key{}
	default:
		srv.logger.Errorf("settings api access: %s: get key by ref: %s", u.ID, err)
		htmlServerError(w, r, err)
		return
	}
	respond(w, "SettingsAPIAccess", map[string]interface{}{
		"User":       u,
		"Key":        k,
		"APIEnabled": srv.APIEnabled,
	})
}

func settingsDeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}
	respond(w, "SettingsDeleteAccount", map[string]interface{}{
		"User":       u,
		"APIEnabled": srv.APIEnabled,
	})
}
