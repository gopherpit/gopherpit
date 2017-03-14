// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func publicEmailSettingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	email, err := emailFromToken(token)
	if err != nil {
		srv.logger.Errorf("public email settings: email from token %s: %s", token, err)
		htmlNotFoundHandler(w, r)
		return
	}

	optedOut, err := srv.NotificationService.IsEmailOptedOut(email)
	if err != nil {
		srv.logger.Errorf("public email settings: is email %s opted-out: %s", email, err)
		htmlServerError(w, r, err)
		return
	}

	respond(w, "PublicEmailSettings", map[string]interface{}{
		"Email":    email,
		"OptedOut": optedOut,
		"Token":    token,
	})
}
