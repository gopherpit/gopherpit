// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpPackages

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"

	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/packages"
)

// Client implements gopherpit.com/gopherpit/services/packages.Service interface.
type Client struct {
	*apiClient.Client
}

// NewClient creates a new Client.
func NewClient(c *apiClient.Client) *Client {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = errorRegistry
	return &Client{Client: c}
}

func (c Client) Domain(ref string) (d *packages.Domain, err error) {
	err = c.JSON("GET", "/domains/"+ref, nil, nil, d)
	err = getServiceError(err)
	return
}

type AddDomainRequest struct {
	Options  *packages.DomainOptions `json:"options"`
	ByUserID string                  `json:"by-user-id"`
}

func (c Client) AddDomain(o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(AddDomainRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = c.JSON("POST", "/domains", nil, bytes.NewReader(body), d)
	err = getServiceError(err)
	return
}

type UpdateDomainRequest struct {
	Options  *packages.DomainOptions `json:"options"`
	ByUserID string                  `json:"by-user-id"`
}

func (c Client) UpdateDomain(ref string, o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(UpdateDomainRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = c.JSON("POST", "/domains/"+ref, nil, bytes.NewReader(body), d)
	err = getServiceError(err)
	return
}

type DeleteDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (c Client) DeleteDomain(ref, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(DeleteDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = c.JSON("DELETE", "/domains/"+ref, nil, bytes.NewReader(body), d)
	err = getServiceError(err)
	return
}

func (c Client) DomainUsers(ref string) (users packages.DomainUsers, err error) {
	err = c.JSON("GET", "/domains/"+ref+"/users", nil, nil, users)
	err = getServiceError(err)
	return
}

type AddUserToDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (c Client) AddUserToDomain(ref, userID, byUserID string) (err error) {
	body, err := json.Marshal(AddUserToDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	err = c.JSON("POST", "/domains/"+ref+"/users/"+userID, nil, bytes.NewReader(body), nil)
	err = getServiceError(err)
	return
}

type RemoveUserToDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (c Client) RemoveUserFromDomain(ref, userID, byUserID string) (err error) {
	body, err := json.Marshal(RemoveUserToDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	err = c.JSON("DELETE", "/domains/"+ref+"/users/"+userID, nil, bytes.NewReader(body), nil)
	err = getServiceError(err)
	return
}

func (c Client) Domains(startRef string, limit int) (page packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/domains", query, nil, page)
	err = getServiceError(err)
	return
}

func (c Client) DomainsByUser(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/domains-by-user/"+userID, query, nil, p)
	err = getServiceError(err)
	return
}

func (c Client) DomainsByOwner(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/domains-by-owner/"+userID, query, nil, p)
	err = getServiceError(err)
	return
}

func (c Client) Package(id string) (p *packages.Package, err error) {
	err = c.JSON("GET", "/packages/"+id, nil, nil, p)
	err = getServiceError(err)
	return
}

type AddPackageRequest struct {
	Options  *packages.PackageOptions `json:"options"`
	ByUserID string                   `json:"by-user-id"`
}

func (c Client) AddPackage(o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(AddPackageRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = c.JSON("POST", "/packages", nil, bytes.NewReader(body), p)
	err = getServiceError(err)
	return
}

type UpdatePackageRequest struct {
	Options  *packages.PackageOptions `json:"options"`
	ByUserID string                   `json:"by-user-id"`
}

func (c Client) UpdatePackage(id string, o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(UpdatePackageRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = c.JSON("POST", "/packages/"+id, nil, bytes.NewReader(body), p)
	err = getServiceError(err)
	return
}

type DeletePackageRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (c Client) DeletePackage(id string, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(DeletePackageRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = c.JSON("DELETE", "/packages/"+id, nil, bytes.NewReader(body), p)
	err = getServiceError(err)
	return
}

func (c Client) PackagesByDomain(domainRef, startName string, limit int) (page packages.PackagesPage, err error) {
	query := url.Values{}
	if startName != "" {
		query.Set("start", startName)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/packages-by-domain/"+domainRef, query, nil, page)
	err = getServiceError(err)
	return
}

func (c Client) ResolvePackage(path string) (resolution *packages.PackageResolution, err error) {
	resolution = &packages.PackageResolution{}
	err = c.JSON("GET", "/paths/"+path, nil, nil, resolution)
	err = getServiceError(err)
	return
}

func (c Client) ChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	record = &packages.ChangelogRecord{}
	err = c.JSON("GET", "/changelogs/"+domainRef+"/record/"+id, nil, nil, record)
	err = getServiceError(err)
	return
}

func (c Client) DeleteChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	record = &packages.ChangelogRecord{}
	err = c.JSON("DELETE", "/changelogs/"+domainRef+"/record/"+id, nil, nil, record)
	err = getServiceError(err)
	return
}

func (c Client) ChangelogForDomain(domainRef, start string, limit int) (changelog packages.Changelog, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/changelogs/"+domainRef, query, nil, changelog)
	err = getServiceError(err)
	return
}
