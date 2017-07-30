// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"resenje.org/jsonresponse"
	"resenje.org/web"
)

type contactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

func contactPrivateFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := contactRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("contact private fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if request.Message == "" {
		srv.Logger.Warningf("contact private fe api: message empty")
		errors.AddFieldError("message", "The message is required.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if err := sendEmailContactEmail(
		u.Email,
		fmt.Sprintf("Contact message from user: %s %s", u.Name, u.Username),
		request.Message,
	); err != nil {
		srv.Logger.Errorf("contact private fe api: %s", err)
		jsonServerError(w, err)
		return
	}
	srv.Logger.Infof("contact private fe api: success %s %s", u.ID, u.Email)

	auditf(r, request, "contact private", "%s: %s", u.ID, u.Email)

	jsonresponse.OK(w, nil)
}

func contactFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	request := contactRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("contact fe api: request decode: %s", err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Email == "" {
		srv.Logger.Warning("contact fe api: request: email empty")
		errors.AddFieldError("email", "E-mail is required.")
	} else {
		emailParts := strings.Split(request.Email, "@")
		if len(emailParts) != 2 {
			srv.Logger.Warning("contact fe api: invalid email %s", request.Email)
			errors.AddFieldError("email", "E-mail address is invalid.")
		} else if _, err := net.ResolveIPAddr("ip", emailParts[1]); err != nil {
			srv.Logger.Warning("contact fe api: invalid email domain %s", request.Email)
			errors.AddFieldError("email", "E-mail address has invalid domain.")
		}
	}
	if request.Message == "" {
		srv.Logger.Warningf("contact fe api: message empty %s", request.Email)
		errors.AddFieldError("message", "The message is required.")
	}
	if request.Name == "" {
		srv.Logger.Warningf("contact fe api: name empty %s", request.Email)
		errors.AddFieldError("name", "Your name is required.")
	}
	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	if err := sendEmailContactEmail(
		request.Email,
		fmt.Sprintf("Contact message from: %s", request.Name),
		request.Message,
	); err != nil {
		srv.Logger.Errorf("contact fe api: %s", err)
		jsonServerError(w, err)
		return
	}
	srv.Logger.Infof("contact fe api: request: success %s", request.Email)

	audit(r, request, "contact", request.Email)

	jsonresponse.OK(w, nil)
}
