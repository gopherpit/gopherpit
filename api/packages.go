// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
)

// Package retrieves a Package instance.
func (c Client) Package(id string) (p Package, err error) {
	err = c.JSON("GET", "/packages/"+id, nil, nil, &p)
	return
}

// AddPackage creates a new Package.
func (c Client) AddPackage(o *PackageOptions) (p Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = c.JSON("POST", "/packages", nil, bytes.NewReader(body), &p)
	return
}

// UpdatePackage updates fields of an existing Package.
func (c Client) UpdatePackage(id string, o *PackageOptions) (p Package, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = c.JSON("POST", "/packages/"+id, nil, bytes.NewReader(body), &p)
	return
}

// DeletePackage removes a Package.
func (c Client) DeletePackage(id string) (p Package, err error) {
	err = c.JSON("DELETE", "/packages/"+id, nil, nil, &p)
	return
}

// DomainPackages retrieves a paginated list of Packages under a domain.
// Values from the previous and next fields in returned page can be provided as
// startRef argument to get a previous or next page in the listing.
func (c Client) DomainPackages(domainRef, start string, limit int) (page PackagesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = c.JSON("GET", "/domains/"+domainRef+"/packages", query, nil, &page)
	return
}
