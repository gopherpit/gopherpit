// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"resenje.org/jsonresponse"
)

func emailOptOutFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	email, err := emailFromToken(token)
	if err != nil {
		srv.Logger.Errorf("email opt-out fe api: token %s: %s", token, err)
		jsonresponse.NotFound(w, nil)
		return
	}

	if !emailRegex.MatchString(email) {
		srv.Logger.Warningf("email opt-out fe api: token %s: invalid data %s", token, email)
		jsonresponse.NotFound(w, nil)
		return
	}

	if err := srv.NotificationService.OptOutEmail(email); err != nil {
		srv.Logger.Errorf("email opt-out fe api: opt-out email %s: %s", email, err)
		jsonServerError(w, err)
		return
	}

	srv.Logger.Infof("email opt-out fe api: success %s", email)

	audit(r, nil, "email opt-out", email)

	jsonresponse.OK(w, nil)
}

func emailRemoveOptOutFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	email, err := emailFromToken(token)
	if err != nil {
		srv.Logger.Errorf("email opt-out remove fe api: token %s: %s", token, err)
		jsonresponse.NotFound(w, nil)
		return
	}

	if !emailRegex.MatchString(email) {
		srv.Logger.Warningf("email opt-out remove fe api: token %s: invalid data %s", token, email)
		jsonresponse.NotFound(w, nil)
		return
	}

	if err := srv.NotificationService.RemoveOptedOutEmail(email); err != nil {
		srv.Logger.Errorf("email opt-out remove fe api: remove email opt-out %s: %s", email, err)
		jsonServerError(w, err)
		return
	}

	srv.Logger.Infof("email opt-out remove fe api: success %s", email)

	audit(r, nil, "email opt-out remove", email)

	jsonresponse.OK(w, nil)
}
