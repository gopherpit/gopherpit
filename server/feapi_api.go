// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"net"
	"net/http"

	"gopherpit.com/gopherpit/services/key"

	"resenje.org/jsonresponse"
	"resenje.org/web"
)

type apiKeyFEAPIResponse struct {
	Secret             string   `json:"secret"`
	AuthorizedNetworks []string `json:"authorized_networks"`
}

func (s *Server) apiKeyFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	k, err := s.KeyService.CreateKey(u.ID, nil)
	if err != nil {
		s.Logger.Errorf("api key fe api: create key %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}

	response := apiKeyFEAPIResponse{
		Secret: k.Secret,
	}

	for _, n := range k.AuthorizedNetworks {
		response.AuthorizedNetworks = append(response.AuthorizedNetworks, n.String())
	}

	s.audit(r, nil, "enable api", "")

	jsonresponse.OK(w, response)
}

func (s *Server) apiKeyDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	err = s.KeyService.DeleteKey(u.ID)
	if err != nil {
		s.Logger.Errorf("api key delete fe api: delete key: %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}

	s.audit(r, nil, "disable api", "")

	jsonresponse.OK(w, nil)
}

type apiRegenerateSecretFEAPIResponse struct {
	Secret string `json:"secret"`
}

func (s *Server) apiRegenerateSecretFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	secret, err := s.KeyService.RegenerateSecret(u.ID)
	if err != nil {
		s.Logger.Errorf("api regenerate secret fe api: regenerate secret %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}

	s.audit(r, nil, "regenerate api secret", "")

	jsonresponse.OK(w, apiRegenerateSecretFEAPIResponse{
		Secret: secret,
	})
}

type apiNetworksFEAPIRequest struct {
	AuthorizedNetworks []string `json:"authorized_networks"`
}

type apiNetworksFEAPIResponse struct {
	AuthorizedNetworks []string `json:"authorized_networks"`
}

func (s *Server) apiNetworksFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := apiNetworksFEAPIRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("api networks fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	authorizedNetworks := []net.IPNet{}
	for _, a := range request.AuthorizedNetworks {
		_, net, err := net.ParseCIDR(a)
		if err != nil {
			s.Logger.Warningf("api networks fe api: user id %s: invalid cidr notation: %s", u.ID, a)
			errors.AddFieldError("authorized_network_"+a, "Invalid CIDR notation.")
			continue
		}
		found := false
		for _, n := range authorizedNetworks {
			if n.String() == net.String() {
				found = true
				break
			}
		}
		if !found {
			authorizedNetworks = append(authorizedNetworks, *net)
		}
	}

	if errors.HasErrors() {
		errors.AddError("Networks are not saved.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	k, err := s.KeyService.UpdateKey(u.ID, &key.Options{
		AuthorizedNetworks: &authorizedNetworks,
	})
	if err != nil {
		if err == key.ErrKeyNotFound {
			s.Logger.Warningf("api networks fe api: update key %s: %s", u.ID, err)
			jsonresponse.NotFound(w, jsonresponse.NewMessage("API key not found"))
			return
		}
		s.Logger.Errorf("api networks fe api: update key %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}

	response := apiNetworksFEAPIResponse{}

	for _, n := range k.AuthorizedNetworks {
		response.AuthorizedNetworks = append(response.AuthorizedNetworks, n.String())
	}

	s.audit(r, response, "update api networks", "")

	jsonresponse.OK(w, response)
}
