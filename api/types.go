// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api // import "gopherpit.com/gopherpit/api"

import apiClient "resenje.org/httputils/client/api"

// MaxLimit is a default maximum number of elements for paged responses.
const MaxLimit = 100

// Domain holds information about GopherPit domain instance.
type Domain struct {
	ID                string `json:"id"`
	FQDN              string `json:"fqdn"`
	OwnerUserID       string `json:"owner_user_id"`
	CertificateIgnore bool   `json:"certificate_ignore,omitempty"`
	Disabled          bool   `json:"disabled,omitempty"`
}

// DomainOptions defines Domain fields that can be changed.
type DomainOptions struct {
	FQDN              *string `json:"fqdn,omitempty"`
	OwnerUserID       *string `json:"owner_user_id,omitempty"`
	CertificateIgnore *bool   `json:"certificate_ignore,omitempty"`
	Disabled          *bool   `json:"disabled,omitempty"`
}

// DomainsPage is a paginated list of Domain instances.
type DomainsPage struct {
	Domains  []Domain `json:"domains"`
	Count    int      `json:"count"`
	Previous string   `json:"previous,omitempty"`
	Next     string   `json:"next,omitempty"`
}

// DomainToken holds information about validation token for a specific fully qualified domain name.
type DomainToken struct {
	FQDN  string `json:"fqdn"`
	Token string `json:"token"`
}

// DomainTokens is an API response with a list of domain tokens.
type DomainTokens struct {
	Tokens []DomainToken `json:"tokens"`
}

// DomainUsers holds information with User IDs who have access to a Domain.
type DomainUsers struct {
	OwnerUserID string   `json:"owner_user_id"`
	UserIDs     []string `json:"user_ids,omitempty"`
}

// VCS is a type that defines possible VCS values for the Package.
type VCS string

// Possible VCS values.
var (
	VCSGit        VCS = "git"
	VCSMercurial  VCS = "hg"
	VCSBazaar     VCS = "bzr"
	VCSSubversion VCS = "svn"
)

// RefType is a type that defines possbile reference type values for the Package.
type RefType string

// Possible reference types.
var (
	RefTypeBranch RefType = "branch"
	RefTypeTag    RefType = "tag"
)

// Package holds data that represents Go package location
// and metadata for remote import path.
// https://golang.org/cmd/go/#hdr-Remote_import_paths
type Package struct {
	ID          string  `json:"id"`
	DomainID    string  `json:"domain_id"`
	FQDN        string  `json:"fqdn"`
	Path        string  `json:"path"`
	VCS         VCS     `json:"vcs"`
	RepoRoot    string  `json:"repo_root"`
	RefType     RefType `json:"ref_type,omitempty"`
	RefName     string  `json:"ref_name,omitempty"`
	GoSource    string  `json:"go_source,omitempty"`
	RedirectURL string  `json:"redirect_url,omitempty"`
	Disabled    bool    `json:"disabled,omitempty"`
}

// PackageOptions defines Package fields that can be changed.
type PackageOptions struct {
	Domain      *string  `json:"domain,omitempty"`
	Path        *string  `json:"path,omitempty"`
	VCS         *VCS     `json:"vcs,omitempty"`
	RepoRoot    *string  `json:"repo_root,omitempty"`
	RefType     *RefType `json:"ref_type"`
	RefName     *string  `json:"ref_name"`
	GoSource    *string  `json:"go_source,omitempty"`
	RedirectURL *string  `json:"redirect_url,omitempty"`
	Disabled    *bool    `json:"disabled,omitempty"`
}

// PackagesPage is a paginated list of Package instances.
type PackagesPage struct {
	Packages []Package `json:"packages"`
	Count    int       `json:"count"`
	Previous string    `json:"previous,omitempty"`
	Next     string    `json:"next,omitempty"`
}

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that the API can return.
var (
	ErrBadRequest                    = errorRegistry.MustAddMessageError(400, "Bad Request")
	ErrUnauthorized                  = errorRegistry.MustAddMessageError(401, "Unauthorized")
	ErrForbidden                     = errorRegistry.MustAddMessageError(403, "Forbidden")
	ErrNotFound                      = errorRegistry.MustAddMessageError(404, "Not Found")
	ErrTooManyRequests               = errorRegistry.MustAddMessageError(429, "Too Many Requests")
	ErrInternalServerError           = errorRegistry.MustAddMessageError(500, "Internal Server Error")
	ErrMaintenance                   = errorRegistry.MustAddMessageError(503, "Maintenance")
	ErrDomainNotFound                = errorRegistry.MustAddMessageError(1000, "Domain Not Found")
	ErrDomainAlreadyExists           = errorRegistry.MustAddMessageError(1001, "Domain Already Exists")
	ErrDomainFQDNRequired            = errorRegistry.MustAddMessageError(1010, "Domain FQDN Required")
	ErrDomainFQDNInvalid             = errorRegistry.MustAddMessageError(1011, "Domain FQDN Invalid")
	ErrDomainNotAvailable            = errorRegistry.MustAddMessageError(1012, "Domain Not Available")
	ErrDomainWithTooManySubdomains   = errorRegistry.MustAddMessageError(1013, "Domain With Too Many Subdomains")
	ErrDomainNeedsVerification       = errorRegistry.MustAddMessageError(1014, "Domain Needs Verification")
	ErrUserDoesNotExist              = errorRegistry.MustAddMessageError(1100, "User Does Not Exist")
	ErrUserAlreadyGranted            = errorRegistry.MustAddMessageError(1101, "User Already Granted")
	ErrUserNotGranted                = errorRegistry.MustAddMessageError(1102, "User Not Granted")
	ErrPackageNotFound               = errorRegistry.MustAddMessageError(2000, "Package Not Found")
	ErrPackageAlreadyExists          = errorRegistry.MustAddMessageError(2001, "Package Already Exists")
	ErrPackageDomainRequired         = errorRegistry.MustAddMessageError(2010, "Package Domain Required")
	ErrPackagePathRequired           = errorRegistry.MustAddMessageError(2020, "Package Path Required")
	ErrPackageVCSRequired            = errorRegistry.MustAddMessageError(2030, "Package VCS Required")
	ErrPackageRepoRootRequired       = errorRegistry.MustAddMessageError(2040, "Package Repository Root Required")
	ErrPackageRepoRootInvalid        = errorRegistry.MustAddMessageError(2041, "Package Repository Root Invalid")
	ErrPackageRepoRootSchemeRequired = errorRegistry.MustAddMessageError(2042, "Package Repository Root Scheme Required")
	ErrPackageRepoRootSchemeInvalid  = errorRegistry.MustAddMessageError(2043, "Package Repository Root Scheme Invalid")
	ErrPackageRepoRootHostInvalid    = errorRegistry.MustAddMessageError(2044, "Package Repository Root Host Invalid")
	ErrPackageRefTypeInvalid         = errorRegistry.MustAddMessageError(2050, "Package Reference Type Invalid")
	ErrPackageRefNameRequired        = errorRegistry.MustAddMessageError(2060, "Package Reference Name Required")
	ErrPackageRefChangeRejected      = errorRegistry.MustAddMessageError(2070, "Package Reference Change Rejected")
	ErrPackageRedirectURLInvalid     = errorRegistry.MustAddMessageError(2080, "Package Redirect URL Invalid")
)
