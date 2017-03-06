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

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/packages"
)

// Service implements gopherpit.com/gopherpit/services/packages.Service interface.
type Service struct {
	Client *apiClient.Client
}

// NewService creates a new Service and injects packages.ErrorRegistry
// in the API Client.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = packages.ErrorRegistry
	return &Service{Client: c}
}

func (s Service) Domain(ref string) (d *packages.Domain, err error) {
	err = s.Client.JSON("GET", "/domains/"+ref, nil, nil, d)
	return
}

type AddDomainRequest struct {
	Options  *packages.DomainOptions `json:"options"`
	ByUserID string                  `json:"by-user-id"`
}

func (s Service) AddDomain(o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(AddDomainRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = s.Client.JSON("POST", "/domains", nil, bytes.NewReader(body), d)
	return
}

type UpdateDomainRequest struct {
	Options  *packages.DomainOptions `json:"options"`
	ByUserID string                  `json:"by-user-id"`
}

func (s Service) UpdateDomain(ref string, o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(UpdateDomainRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = s.Client.JSON("POST", "/domains/"+ref, nil, bytes.NewReader(body), d)
	return
}

type DeleteDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (s Service) DeleteDomain(ref, byUserID string) (d *packages.Domain, err error) {
	body, err := json.Marshal(DeleteDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	d = &packages.Domain{}
	err = s.Client.JSON("DELETE", "/domains/"+ref, nil, bytes.NewReader(body), d)
	return
}

func (s Service) DomainUsers(ref string) (users packages.DomainUsers, err error) {
	err = s.Client.JSON("GET", "/domains/"+ref+"/users", nil, nil, users)
	return
}

type AddUserToDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (s Service) AddUserToDomain(ref, userID, byUserID string) (err error) {
	body, err := json.Marshal(AddUserToDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/domains/"+ref+"/users/"+userID, nil, bytes.NewReader(body), nil)
	return
}

type RemoveUserToDomainRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (s Service) RemoveUserFromDomain(ref, userID, byUserID string) (err error) {
	body, err := json.Marshal(RemoveUserToDomainRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("DELETE", "/domains/"+ref+"/users/"+userID, nil, bytes.NewReader(body), nil)
	return
}

func (s Service) Domains(startRef string, limit int) (page packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/domains", query, nil, page)
	return
}

func (s Service) DomainsByUser(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/domains-by-user/"+userID, query, nil, p)
	return
}

func (s Service) DomainsByOwner(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	query := url.Values{}
	if startRef != "" {
		query.Set("start", startRef)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/domains-by-owner/"+userID, query, nil, p)
	return
}

func (s Service) Package(id string) (p *packages.Package, err error) {
	err = s.Client.JSON("GET", "/packages/"+id, nil, nil, p)
	return
}

type AddPackageRequest struct {
	Options  *packages.PackageOptions `json:"options"`
	ByUserID string                   `json:"by-user-id"`
}

func (s Service) AddPackage(o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(AddPackageRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = s.Client.JSON("POST", "/packages", nil, bytes.NewReader(body), p)
	return
}

type UpdatePackageRequest struct {
	Options  *packages.PackageOptions `json:"options"`
	ByUserID string                   `json:"by-user-id"`
}

func (s Service) UpdatePackage(id string, o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(UpdatePackageRequest{
		Options:  o,
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = s.Client.JSON("POST", "/packages/"+id, nil, bytes.NewReader(body), p)
	return
}

type DeletePackageRequest struct {
	ByUserID string `json:"by-user-id"`
}

func (s Service) DeletePackage(id string, byUserID string) (p *packages.Package, err error) {
	body, err := json.Marshal(DeletePackageRequest{
		ByUserID: byUserID,
	})
	if err != nil {
		return
	}
	p = &packages.Package{}
	err = s.Client.JSON("DELETE", "/packages/"+id, nil, bytes.NewReader(body), p)
	return
}

func (s Service) PackagesByDomain(domainRef, startName string, limit int) (page packages.PackagesPage, err error) {
	query := url.Values{}
	if startName != "" {
		query.Set("start", startName)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/packages-by-domain/"+domainRef, query, nil, page)
	return
}

func (s Service) ResolvePackage(path string) (resolution *packages.PackageResolution, err error) {
	resolution = &packages.PackageResolution{}
	err = s.Client.JSON("GET", "/paths/"+path, nil, nil, resolution)
	return
}

func (s Service) ChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	record = &packages.ChangelogRecord{}
	err = s.Client.JSON("GET", "/changelogs/"+domainRef+"/record/"+id, nil, nil, record)
	return
}

func (s Service) DeleteChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	record = &packages.ChangelogRecord{}
	err = s.Client.JSON("DELETE", "/changelogs/"+domainRef+"/record/"+id, nil, nil, record)
	return
}

func (s Service) ChangelogForDomain(domainRef, start string, limit int) (changelog packages.Changelog, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/changelogs/"+domainRef, query, nil, changelog)
	return
}
