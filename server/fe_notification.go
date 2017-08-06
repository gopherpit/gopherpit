// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Server) publicEmailSettingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	email, err := s.emailFromToken(token)
	if err != nil {
		s.Logger.Errorf("public email settings: email from token %s: %s", token, err)
		s.htmlServerError(w, r, err)
		return
	}

	optedOut, err := s.NotificationService.IsEmailOptedOut(email)
	if err != nil {
		s.Logger.Errorf("public email settings: is email %s opted-out: %s", email, err)
		s.htmlServerError(w, r, err)
		return
	}

	s.html.Respond(w, "PublicEmailSettings", map[string]interface{}{
		"Email":    email,
		"OptedOut": optedOut,
		"Token":    token,
	})
}
