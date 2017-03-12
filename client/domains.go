// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"

	"gopherpit.com/gopherpit/api"
)

// Domain retrieves a Domain instance.
func (s Service) Domain(ref string) (d *api.Domain, err error) {
	d = &api.Domain{}
	err = s.Client.JSON("GET", "/domains/"+ref, nil, nil, d)
	return
}

// AddDomain creates a new Domain.
func (s Service) AddDomain(o *api.DomainOptions) (d *api.Domain, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Domain{}
	err = s.Client.JSON("POST", "/domains", nil, bytes.NewReader(body), d)
	return
}

// UpdateDomain creates a new Domain.
func (s Service) UpdateDomain(ref string, o *api.DomainOptions) (d *api.Domain, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Domain{}
	err = s.Client.JSON("POST", "/domains/"+ref, nil, bytes.NewReader(body), d)
	return
}

// DeleteDomain removes a Domain.
func (s Service) DeleteDomain(ref string) (d *api.Domain, err error) {
	d = &api.Domain{}
	err = s.Client.JSON("DELETE", "/domains/"+ref, nil, nil, d)
	return
}

// Domains retrieves a paginated list of Domains.
func (s Service) Domains(startRef string, limit int) (page api.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/domains/", query, nil, page)
	return
}

// DomainTokens retrieves a list of validation tokens for domain.
func (s Service) DomainTokens(fqdn string) (tokens api.DomainTokens, err error) {
	err = s.Client.JSON("GET", "/domains/"+fqdn+"/tokens", nil, nil, tokens)
	return
}

// DomainUsers retrieves a list of user IDs that have write access to
// domain packages and domain owner user ID.
func (s Service) DomainUsers(ref string) (users api.DomainUsers, err error) {
	err = s.Client.JSON("GET", "/domains/"+ref+"/users", nil, nil, users)
	return
}

// GrantDomainUser gives write access to domain packages for a user.
func (s Service) GrantDomainUser(ref, user string) error {
	return s.Client.JSON("POST", "/domains/"+ref+"/users/"+user, nil, nil, nil)
}

// RevokeDomainUser removes write access to domain packages for a user.
func (s Service) RevokeDomainUser(ref, user string) error {
	return s.Client.JSON("DELETE", "/domains/"+ref+"/users/"+user, nil, nil, nil)
}
