// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"

	"gopherpit.com/gopherpit/services/user"
)

func (s Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	r, err := s.logout(w, r)
	if err != nil {
		s.logger.Errorf("logout: %s", err)
		s.htmlServerError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s Server) registrationHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, s.templateRegistration(), nil)
}

func (s Server) passwordResetHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, s.templatePasswordReset(), map[string]interface{}{
		"Token": mux.Vars(r)["token"],
	})
}

func (s Server) passwordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, s.templatePasswordResetToken(), nil)
}

func (s Server) emailValidationHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	vars := mux.Vars(r)
	token := vars["token"]

	u2, err := s.UserService.ChangeEmail(u.ID, token)
	if err != nil {
		if terr, ok := err.(*user.Error); ok {
			s.logger.Warningf("email validation: user %s: change email token %s: %s", u.ID, token, terr)
			respond(w, s.templateEmailValidation(), map[string]interface{}{
				"Valid": false,
				"User":  u,
			})
			return
		}
		s.logger.Errorf("email validation: user %s: change email token %s: %s", u.ID, token, err)
		s.htmlServerError(w, r, err)
		return
	}
	respond(w, s.templateEmailValidation(), map[string]interface{}{
		"Valid": !u2.EmailUnvalidated,
		"User":  u2,
	})
}

func (s Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	respond(w, s.templateSettings(), map[string]interface{}{
		"User": u,
	})
}

func (s Server) settingsEmailHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := s.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		s.logger.Errorf("settings email: %s: is email opted out api: %s", u.Email, err)
		s.htmlServerError(w, r, err)
		return
	}
	respond(w, s.templateSettingsEmail(), map[string]interface{}{
		"User":     u,
		"OptedOut": optedOut,
	})
}

func (s Server) settingsNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := s.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		s.logger.Errorf("settings notifications: %s: is email opted out api: %s", u.Email, err)
		s.htmlServerError(w, r, err)
		return
	}
	respond(w, s.templateSettingsNotifications(), map[string]interface{}{
		"User":     u,
		"OptedOut": optedOut,
	})
}

func (s Server) settingsPasswordHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	respond(w, s.templateSettingsPassword(), map[string]interface{}{
		"User": u,
	})
}

func (s Server) settingsDeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}
	respond(w, s.templateSettingsDeleteAccount(), map[string]interface{}{
		"User": u,
	})
}
