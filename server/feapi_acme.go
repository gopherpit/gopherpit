// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"net/http"

	"resenje.org/httputils"
	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/services/certificate"
)

type registerACMEUserRequest struct {
	Email     string `json:"email"`
	Directory string `json:"directory"`
}

func (s Server) registerACMEUserFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	au, err := s.CertificateService.ACMEUser()
	if err != nil && err != certificate.ACMEUserNotFound {
		s.logger.Warningf("register acme user fe api: acme user: %s", err)
		jsonServerError(w, err)
		return
	}
	if au != nil {
		jsonresponse.Forbidden(w, nil)
		return
	}

	request := registerACMEUserRequest{}
	errors := httputils.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("register acme user fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	var directoryURL string
	switch request.Directory {
	case "":
		s.logger.Warningf("register acme user fe api: directory empty")
		errors.AddFieldError("directory", "Directory is required.")
	case "production":
		directoryURL = s.ACMEDirectoryURL
	case "staging":
		directoryURL = s.ACMEDirectoryURLStaging
	default:
		s.logger.Warningf("register acme user fe api: directory invalid: %s", request.Directory)
		errors.AddFieldError("directory", "Directory is not valid.")
	}

	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	au, err = s.CertificateService.RegisterACMEUser(directoryURL, request.Email)
	if err != nil {
		if err == certificate.ACMEUserEmailInvalid {
			errors.AddFieldError("email", "E-mail address is invalid.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		s.logger.Warningf("register acme user fe api: register acme user: %s", err)
		jsonServerError(w, err)
		return
	}

	s.logger.Infof("register acme user fe api: success %d %s", au.ID, au.Email)

	s.auditf(r, nil, "register acme user", "%d: %s", au.ID, au.Email)

	jsonresponse.OK(w, nil)
}
