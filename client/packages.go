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
func (c Client) Package(id string) (d *api.Package, err error) {
	d = &api.Package{}
	err = c.Client.JSON("GET", "/packages/"+id, nil, nil, d)
	return
}

// AddPackage creates a new Package.
func (c Client) AddPackage(o *api.PackageOptions) (d *api.Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Package{}
	err = c.Client.JSON("POST", "/packages", nil, bytes.NewReader(body), d)
	return
}

// UpdatePackage creates a new Package.
func (c Client) UpdatePackage(id string, o *api.PackageOptions) (d *api.Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	d = &api.Package{}
	err = c.Client.JSON("POST", "/packages/"+id, nil, bytes.NewReader(body), d)
	return
}

// DeletePackage removes a Package.
func (c Client) DeletePackage(id string) (d *api.Package, err error) {
	d = &api.Package{}
	err = c.Client.JSON("DELETE", "/packages/"+id, nil, nil, d)
	return
}

// DomainPackages retrieves a paginated list of Packages under a domain.
func (c Client) DomainPackages(domainRef, start string, limit int) (page api.PackagesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.Client.JSON("GET", "/domain/"+domainRef+"/packages", query, nil, &page)
	return
}
