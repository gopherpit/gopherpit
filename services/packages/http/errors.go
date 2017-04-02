// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpPackages

import (
	"gopherpit.com/gopherpit/services/packages"
	"resenje.org/httputils/client/api"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrForbidden                     = errorRegistry.MustAddMessageError(403, "Forbidden")
	ErrDomainNotFound                = errorRegistry.MustAddMessageError(1000, "Domain Not Found")
	ErrDomainAlreadyExists           = errorRegistry.MustAddMessageError(1001, "Domain Already Exists")
	ErrDomainFQDNRequired            = errorRegistry.MustAddMessageError(1010, "Domain FQDN Required")
	ErrDomainOwnerUserIDRequired     = errorRegistry.MustAddMessageError(1020, "Domain Owner User ID Required")
	ErrUserDoesNotExist              = errorRegistry.MustAddMessageError(1100, "User Does Not Exist")
	ErrUserExists                    = errorRegistry.MustAddMessageError(1101, "User Exists")
	ErrPackageNotFound               = errorRegistry.MustAddMessageError(2000, "Package Not Found")
	ErrPackageAlreadyExists          = errorRegistry.MustAddMessageError(2001, "Package Already Exists")
	ErrPackageDomainRequired         = errorRegistry.MustAddMessageError(2010, "Package Domain Required")
	ErrPackagePathRequired           = errorRegistry.MustAddMessageError(2011, "Package Path Required")
	ErrPackageVCSRequired            = errorRegistry.MustAddMessageError(2012, "Package VCS Required")
	ErrPackageRepoRootRequired       = errorRegistry.MustAddMessageError(2040, "Package Repository Root Required")
	ErrPackageRepoRootInvalid        = errorRegistry.MustAddMessageError(2041, "Package Repository Root Invalid")
	ErrPackageRepoRootSchemeRequired = errorRegistry.MustAddMessageError(2042, "Package Repository Root Scheme Required")
	ErrPackageRepoRootSchemeInvalid  = errorRegistry.MustAddMessageError(2043, "Package Repository Root Scheme Invalid")
	ErrPackageRepoRootHostInvalid    = errorRegistry.MustAddMessageError(2044, "Package Repository Root Host Invalid")
	ErrPackageRefChangeRejected      = errorRegistry.MustAddMessageError(2070, "Package Reference Change Rejected")
	ErrChangelogRecordNotFound       = errorRegistry.MustAddMessageError(3000, "Changelog Record Not Found")
)

var errorMap = map[error]error{
	ErrForbidden:                     packages.ErrForbidden,
	ErrDomainNotFound:                packages.ErrDomainNotFound,
	ErrDomainAlreadyExists:           packages.ErrDomainAlreadyExists,
	ErrDomainFQDNRequired:            packages.ErrDomainFQDNRequired,
	ErrDomainOwnerUserIDRequired:     packages.ErrDomainOwnerUserIDRequired,
	ErrUserDoesNotExist:              packages.ErrUserDoesNotExist,
	ErrUserExists:                    packages.ErrUserExists,
	ErrPackageNotFound:               packages.ErrPackageNotFound,
	ErrPackageAlreadyExists:          packages.ErrPackageAlreadyExists,
	ErrPackageDomainRequired:         packages.ErrPackageDomainRequired,
	ErrPackagePathRequired:           packages.ErrPackagePathRequired,
	ErrPackageVCSRequired:            packages.ErrPackageVCSRequired,
	ErrPackageRepoRootRequired:       packages.ErrPackageRepoRootRequired,
	ErrPackageRepoRootInvalid:        packages.ErrPackageRepoRootInvalid,
	ErrPackageRepoRootSchemeRequired: packages.ErrPackageRepoRootSchemeRequired,
	ErrPackageRepoRootSchemeInvalid:  packages.ErrPackageRepoRootSchemeInvalid,
	ErrPackageRepoRootHostInvalid:    packages.ErrPackageRepoRootHostInvalid,
	ErrPackageRefChangeRejected:      packages.ErrPackageRefChangeRejected,
	ErrChangelogRecordNotFound:       packages.ErrChangelogRecordNotFound,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
