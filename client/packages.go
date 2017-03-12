// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// Package retrieves a Package instance.
func (s Service) Package(id string) (d *api.Package, err error) {
	d = &api.Package{}
	err = s.Client.JSON("GET", "/packages/"+id, nil, nil, d)
	return
}

// AddPackage creates a new Package.
func (s Service) AddPackage(o *api.PackageOptions) (d *api.Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Package{}
	err = s.Client.JSON("POST", "/packages", nil, bytes.NewReader(body), d)
	return
}

// UpdatePackage creates a new Package.
func (s Service) UpdatePackage(id string, o *api.PackageOptions) (d *api.Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Package{}
	err = s.Client.JSON("POST", "/packages/"+id, nil, bytes.NewReader(body), d)
	return
}

// DeletePackage removes a Package.
func (s Service) DeletePackage(id string) (d *api.Package, err error) {
	d = &api.Package{}
	err = s.Client.JSON("DELETE", "/packages/"+id, nil, nil, d)
	return
}

// DomainPackages retrieves a paginated list of Packages under a domain.
func (s Service) DomainPackages(domainRef, start string, limit int) (page api.PackagesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/domain/"+domainRef+"/packages", query, nil, page)
	return
}
