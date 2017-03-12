// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/services/packages"
)

func packagesDomainToAPIDomain(d packages.Domain) api.Domain {
	return api.Domain{
		ID:                d.ID,
		FQDN:              d.FQDN,
		OwnerUserID:       d.OwnerUserID,
		CertificateIgnore: d.CertificateIgnore,
		Disabled:          d.Disabled,
	}
}

func packagesPackageToAPIPackage(p packages.Package, d *packages.Domain) api.Package {
	if d == nil {
		d = p.Domain
	}
	return api.Package{
		ID:          p.ID,
		DomainID:    d.ID,
		FQDN:        d.FQDN,
		Path:        p.Path,
		VCS:         api.VCS(p.VCS),
		RepoRoot:    p.RepoRoot,
		RefType:     p.RefType,
		RefName:     p.RefName,
		GoSource:    p.GoSource,
		RedirectURL: p.RedirectURL,
		Disabled:    p.Disabled,
	}
}
