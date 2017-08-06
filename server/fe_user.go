// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"

	"gopherpit.com/gopherpit/services/key"
)

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	r, err := s.logout(w, r)
	if err != nil {
		s.Logger.Errorf("logout: %s", err)
		s.htmlServerError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) registrationHandler(w http.ResponseWriter, r *http.Request) {
	s.html.Respond(w, "Registration", nil)
}

func (s *Server) passwordResetHandler(w http.ResponseWriter, r *http.Request) {
	s.html.Respond(w, "PasswordReset", map[string]interface{}{
		"Token": mux.Vars(r)["token"],
	})
}

func (s *Server) passwordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	s.html.Respond(w, "PasswordResetToken", nil)
}

func (s *Server) emailValidationHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	vars := mux.Vars(r)
	token := vars["token"]

	u2, err := s.UserService.ChangeEmail(u.ID, token)
	if err != nil {
		s.Logger.Errorf("email validation: user %s: change email token %s: %s", u.ID, token, err)
		s.html.Respond(w, "EmailValidation", map[string]interface{}{
			"Valid": false,
			"User":  u,
		})
		return
	}
	s.html.Respond(w, "EmailValidation", map[string]interface{}{
		"Valid": !u2.EmailUnvalidated,
		"User":  u2,
	})
}

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	s.html.Respond(w, "Settings", map[string]interface{}{
		"User":       u,
		"APIEnabled": s.APIEnabled,
	})
}

func (s *Server) settingsEmailHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := s.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		s.Logger.Errorf("settings email: %s: is email opted out api: %s", u.Email, err)
		s.htmlServerError(w, r, err)
		return
	}
	s.html.Respond(w, "SettingsEmail", map[string]interface{}{
		"User":       u,
		"OptedOut":   optedOut,
		"APIEnabled": s.APIEnabled,
	})
}

func (s *Server) settingsNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	optedOut, err := s.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		s.Logger.Errorf("settings notifications: %s: is email opted out api: %s", u.Email, err)
		s.htmlServerError(w, r, err)
		return
	}
	s.html.Respond(w, "SettingsNotifications", map[string]interface{}{
		"User":       u,
		"OptedOut":   optedOut,
		"APIEnabled": s.APIEnabled,
	})
}

func (s *Server) settingsPasswordHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	s.html.Respond(w, "SettingsPassword", map[string]interface{}{
		"User":       u,
		"APIEnabled": s.APIEnabled,
	})
}

func (s *Server) apiAccessSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.APIEnabled {
		s.htmlNotFoundHandler(w, r)
		return
	}
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	k, err := s.KeyService.KeyByRef(u.ID)
	switch err {
	case nil:
	case key.ErrKeyNotFound:
		k = &key.Key{}
	default:
		s.Logger.Errorf("settings api access: %s: get key by ref: %s", u.ID, err)
		s.htmlServerError(w, r, err)
		return
	}
	s.html.Respond(w, "SettingsAPIAccess", map[string]interface{}{
		"User":       u,
		"Key":        k,
		"APIEnabled": s.APIEnabled,
	})
}

func (s *Server) settingsDeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}
	s.html.Respond(w, "SettingsDeleteAccount", map[string]interface{}{
		"User":       u,
		"APIEnabled": s.APIEnabled,
	})
}
