// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

const MaxLimit = 100

type Domain struct {
	ID                string `json:"id"`
	FQDN              string `json:"fqdn"`
	OwnerUserID       string `json:"owner_user_id,omitempty"`
	CertificateIgnore bool   `json:"certificate_ignore,omitempty"`
	Disabled          bool   `json:"disabled,omitempty"`
}

type DomainOptions struct {
	FQDN              *string `json:"fqdn,omitempty"`
	OwnerUserID       *string `json:"owner_user_id,omitempty"`
	CertificateIgnore *bool   `json:"certificate_ignore,omitempty"`
	Disabled          *bool   `json:"disabled,omitempty"`
}

type Domains []Domain

type DomainsPage struct {
	Domains  Domains `json:"domains"`
	Count    int     `json:"count"`
	Previous string  `json:"previous,omitempty"`
	Next     string  `json:"next,omitempty"`
}

type DomainToken struct {
	Domain string `json:"domain"`
	Token  string `json:"token"`
}

type DomainTokens struct {
	Tokens []DomainToken `json:"tokens"`
}

type DomainUsers struct {
	OwnerUserID string   `json:"owner_user_id"`
	UserIDs     []string `json:"user_ids,omitempty"`
}

type VCS string

var (
	VCSGit        VCS = "git"
	VCSMercurial  VCS = "hg"
	VCSBazaar     VCS = "bzr"
	VCSSubversion VCS = "svn"
)

// Package holds data that represents Go package location
// and metadate for remote import path.
// https://golang.org/cmd/go/#hdr-Remote_import_paths
type Package struct {
	ID          string `json:"id"`
	DomainID    string `json:"domain_id"`
	FQDN        string `json:"fqdn"`
	Path        string `json:"path"`
	VCS         VCS    `json:"vcs"`
	RepoRoot    string `json:"repo_root"`
	RefType     string `json:"ref_type"`
	RefName     string `json:"ref_name"`
	GoSource    string `json:"go_source,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type PackageOptions struct {
	Domain      *string `json:"domain,omitempty"`
	Path        *string `json:"path,omitempty"`
	VCS         *VCS    `json:"vcs,omitempty"`
	RepoRoot    *string `json:"repo_root,omitempty"`
	RefType     *string `json:"ref_type"`
	RefName     *string `json:"ref_name"`
	GoSource    *string `json:"go_source,omitempty"`
	RedirectURL *string `json:"redirect_url,omitempty"`
	Disabled    *bool   `json:"disabled,omitempty"`
}

type Packages []Package

type PackagesPage struct {
	Packages Packages `json:"packages"`
	Domain   *Domain  `json:"domain,omitempty"`
	Count    int      `json:"count"`
	Previous string   `json:"previous,omitempty"`
	Next     string   `json:"next,omitempty"`
}

var (
	Forbidden                     = NewError(403, "Forbidden")
	Maintenance                   = NewError(503, "Maintenance")
	DomainNotFound                = NewError(1000, "Domain Not Found")
	DomainAlreadyExists           = NewError(1001, "Domain Already Exists")
	DomainFQDNRequired            = NewError(1010, "Domain FQDN Required")
	DomainFQDNInvalid             = NewError(1011, "Domain FQDN Invalid")
	DomainNotAvailable            = NewError(1012, "Domain Not Available")
	DomainWithTooManySubdomains   = NewError(1013, "Domain With Too Many Subdomains")
	DomainNeedsVerification       = NewError(1014, "Domain Needs Verification")
	UserDoesNotExist              = NewError(1100, "User Does Not Exist")
	UserAlreadyGranted            = NewError(1101, "User Already Granted")
	UserNotGranted                = NewError(1102, "User Not Granted")
	PackageNotFound               = NewError(2000, "Package Not Found")
	PackageAlreadyExists          = NewError(2001, "Package Already Exists")
	PackageDomainRequired         = NewError(2010, "Package Domain Required")
	PackagePathRequired           = NewError(2020, "Package Path Required")
	PackageVCSRequired            = NewError(2030, "Package VCS Required")
	PackageRepoRootRequired       = NewError(2040, "Package Repository Root Required")
	PackageRepoRootInvalid        = NewError(2041, "Package Repository Root Invalid")
	PackageRepoRootSchemeRequired = NewError(2042, "Package Repository Root Scheme Required")
	PackageRepoRootSchemeInvalid  = NewError(2043, "Package Repository Root Scheme Invalid")
	PackageRepoRootHostInvalid    = NewError(2044, "Package Repository Root Host Invalid")
	PackageRefTypeInvalid         = NewError(2050, "Package Reference Type Invalid")
	PackageRefNameRequired        = NewError(2060, "Package Reference Name Required")
	PackageRefChangeRejected      = NewError(2070, "Package Reference Change Rejected")
	PackageRedirectURLInvalid     = NewError(2080, "Package Redirect URL Invalid")
)
