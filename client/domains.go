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
func (c Client) Domain(ref string) (d *api.Domain, err error) {
	d = &api.Domain{}
	err = c.Client.JSON("GET", "/domains/"+ref, nil, nil, d)
	return
}

// AddDomain creates a new Domain.
func (c Client) AddDomain(o *api.DomainOptions) (d *api.Domain, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Domain{}
	err = c.Client.JSON("POST", "/domains", nil, bytes.NewReader(body), d)
	return
}

// UpdateDomain creates a new Domain.
func (c Client) UpdateDomain(ref string, o *api.DomainOptions) (d *api.Domain, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Domain{}
	err = c.Client.JSON("POST", "/domains/"+ref, nil, bytes.NewReader(body), d)
	return
}

// DeleteDomain removes a Domain.
func (c Client) DeleteDomain(ref string) (d *api.Domain, err error) {
	d = &api.Domain{}
	err = c.Client.JSON("DELETE", "/domains/"+ref, nil, nil, d)
	return
}

// Domains retrieves a paginated list of Domains.
func (c Client) Domains(startRef string, limit int) (page api.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.Client.JSON("GET", "/domains", query, nil, &page)
	return
}

// DomainTokens retrieves a list of validation tokens for domain.
func (c Client) DomainTokens(fqdn string) (tokens api.DomainTokens, err error) {
	err = c.Client.JSON("GET", "/domains/"+fqdn+"/tokens", nil, nil, &tokens)
	return
}

// DomainUsers retrieves a list of user IDs that have write access to
// domain packages and domain owner user ID.
func (c Client) DomainUsers(ref string) (users api.DomainUsers, err error) {
	err = c.Client.JSON("GET", "/domains/"+ref+"/users", nil, nil, &users)
	return
}

// GrantDomainUser gives write access to domain packages for a user.
func (c Client) GrantDomainUser(ref, user string) error {
	return c.Client.JSON("POST", "/domains/"+ref+"/users/"+user, nil, nil, nil)
}

// RevokeDomainUser removes write access to domain packages for a user.
func (c Client) RevokeDomainUser(ref, user string) error {
	return c.Client.JSON("DELETE", "/domains/"+ref+"/users/"+user, nil, nil, nil)
}
