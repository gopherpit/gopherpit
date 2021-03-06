// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packages

import (
	"errors"
	"time"
)

type Service interface {
	DomainService
	PackageService
	ChangelogService
}

type DomainService interface {
	Domain(ref string) (*Domain, error)
	AddDomain(o *DomainOptions, byUserID string) (*Domain, error)
	UpdateDomain(ref string, o *DomainOptions, byUserID string) (*Domain, error)
	DeleteDomain(ref, byUserID string) (*Domain, error)
	DomainUsers(ref string) (DomainUsers, error)
	AddUserToDomain(ref, userID, byUserID string) error
	RemoveUserFromDomain(ref, userID, byUserID string) error
	Domains(startFQDN string, limit int) (DomainsPage, error)
	DomainsByUser(userID, startRef string, limit int) (DomainsPage, error)
	DomainsByOwner(userID, startRef string, limit int) (DomainsPage, error)
}

type PackageService interface {
	Package(id string) (*Package, error)
	AddPackage(o *PackageOptions, byUserID string) (*Package, error)
	UpdatePackage(id string, o *PackageOptions, byUserID string) (*Package, error)
	DeletePackage(id string, byUserID string) (*Package, error)
	PackagesByDomain(domainRef, startName string, limit int) (PackagesPage, error)
	ResolvePackage(path string) (*PackageResolution, error)
}

type ChangelogService interface {
	ChangelogRecord(domainRef, id string) (*ChangelogRecord, error)
	DeleteChangelogRecord(domainRef, id string) (*ChangelogRecord, error)
	ChangelogForDomain(domainRef, start string, limit int) (Changelog, error)
}

type Domain struct {
	ID                string `json:"id"`
	FQDN              string `json:"fqdn"`
	OwnerUserID       string `json:"owner-user-id,omitempty"`
	CertificateIgnore bool   `json:"certificate-ignore,omitempty"`
	Disabled          bool   `json:"disabled,omitempty"`

	// Internal functionality fields (changes not logged)
	CertificateIgnoreMissing bool `json:"certificate-ignore-missing,omitempty"`
}

type DomainOptions struct {
	FQDN              *string `json:"fqdn,omitempty"`
	OwnerUserID       *string `json:"owner-user-id,omitempty"`
	CertificateIgnore *bool   `json:"certificate-ignore,omitempty"`
	Disabled          *bool   `json:"disabled,omitempty"`

	CertificateIgnoreMissing *bool `json:"certificate-ignore-missing,omitempty"`
}

type Domains []Domain

type DomainsPage struct {
	Domains  Domains `json:"domains"`
	UserID   string  `json:"user-id,omitempty"`
	Previous string  `json:"previous,omitempty"`
	Next     string  `json:"next,omitempty"`
	Count    int     `json:"count,omitempty"`
}

type DomainUsers struct {
	OwnerUserID string   `json:"owner-user-id"`
	UserIDs     []string `json:"user-ids,omitempty"`
}

type VCS string

var (
	VCSGit        VCS = "git"
	VCSMercurial  VCS = "hg"
	VCSBazaar     VCS = "bzr"
	VCSSubversion VCS = "svn"
)

var VCSSchemes = map[VCS][]string{
	VCSGit:        {"https", "http", "git", "git+ssh", "ssh"},
	VCSMercurial:  {"https", "http", "ssh"},
	VCSBazaar:     {"https", "http", "bzr", "bzr+ssh"},
	VCSSubversion: {"https", "http", "svn", "svn+ssh"},
}

type RefType string

const (
	RefTypeBranch RefType = "branch"
	RefTypeTag    RefType = "tag"
)

// Package hods data that represents Go package location
// and metadate for remotr import path.
// https://golang.org/cmd/go/#hdr-Remote_import_paths
type Package struct {
	ID          string  `json:"id"`
	Domain      *Domain `json:"domain,omitempty"`
	Path        string  `json:"path"`
	VCS         VCS     `json:"vcs"`
	RepoRoot    string  `json:"repo-root"`
	RefType     RefType `json:"ref-type"`
	RefName     string  `json:"ref-name"`
	GoSource    string  `json:"go-source,omitempty"`
	RedirectURL string  `json:"redirect-url,omitempty"`
	Disabled    bool    `json:"disabled,omitempty"`
}

func (p Package) ImportPrefix() string {
	return p.Domain.FQDN + p.Path
}

type PackageOptions struct {
	Domain      *string  `json:"domain,omitempty"`
	Path        *string  `json:"path,omitempty"`
	VCS         *VCS     `json:"vcs,omitempty"`
	RepoRoot    *string  `json:"repo-root,omitempty"`
	RefType     *RefType `json:"ref-type"`
	RefName     *string  `json:"ref-name"`
	GoSource    *string  `json:"go-source,omitempty"`
	RedirectURL *string  `json:"redirect-url,omitempty"`
	Disabled    *bool    `json:"disabled,omitempty"`
}

type Packages []Package

type PackagesPage struct {
	Packages Packages `json:"packages"`
	Domain   *Domain  `json:"domain,omitempty"`
	Previous string   `json:"previous,omitempty"`
	Next     string   `json:"next,omitempty"`
	Count    int      `json:"count,omitempty"`
}

type PackageResolution struct {
	ImportPrefix string  `json:"import-prefix"`
	VCS          VCS     `json:"vcs"`
	RepoRoot     string  `json:"repo-root"`
	RefType      RefType `json:"ref-type"`
	RefName      string  `json:"ref-name"`
	GoSource     string  `json:"go-source,omitempty"`
	RedirectURL  string  `json:"redirect-url,omitempty"`
	Disabled     bool    `json:"disabled,omitempty"`
}

type Action string

var (
	ActionAddDomain        Action = "add-domain"
	ActionUpdateDomain     Action = "update-domain"
	ActionDeleteDomain     Action = "delete-domain"
	ActionDomainAddUser    Action = "domain-add-user"
	ActionDomainRemoveUser Action = "domain-remove-user"
	ActionAddPackage       Action = "add-package"
	ActionUpdatePackage    Action = "update-package"
	ActionDeletePackage    Action = "delete-package"
)

type Change struct {
	Field string  `json:"field"`
	From  *string `json:"from,omitempty"`
	To    *string `json:"to,omitempty"`
}

func (c Change) ToString() string {
	if c.To == nil {
		return ""
	}
	return *c.To
}

func (c Change) FromString() string {
	if c.From == nil {
		return ""
	}
	return *c.From
}

type Changes []Change

type ChangelogRecord struct {
	ID        string    `json:"id,omitempty"`
	Time      time.Time `json:"time,omitempty"`
	DomainID  string    `json:"domain-id,omitempty"`
	FQDN      string    `json:"fqdn,omitempty"`
	PackageID string    `json:"package-id,omitempty"`
	Path      string    `json:"path,omitempty"`
	UserID    string    `json:"user-id,omitempty"`
	Action    Action    `json:"action,omitempty"`
	Changes   Changes   `json:"changes,omitempty"`
}

func (c ChangelogRecord) ImportPrefix() string {
	if c.FQDN == "" || c.Path == "" {
		return ""
	}
	return c.FQDN + c.Path
}

type ChangelogRecords []ChangelogRecord

type Changelog struct {
	Records  ChangelogRecords `json:"records"`
	Previous string           `json:"previous,omitempty"`
	Next     string           `json:"next,omitempty"`
	Count    int              `json:"count,omitempty"`
}

// Errors that are related to the Packages Service.
var (
	ErrForbidden                     = errors.New("forbidden")
	ErrDomainNotFound                = errors.New("domain not found")
	ErrDomainAlreadyExists           = errors.New("domain already exists")
	ErrDomainFQDNRequired            = errors.New("domain fqdn required")
	ErrDomainOwnerUserIDRequired     = errors.New("domain owner user id required")
	ErrUserDoesNotExist              = errors.New("user does not exist")
	ErrUserExists                    = errors.New("user exists")
	ErrPackageNotFound               = errors.New("package not found")
	ErrPackageAlreadyExists          = errors.New("package already exists")
	ErrPackageDomainRequired         = errors.New("package domain required")
	ErrPackagePathRequired           = errors.New("package path required")
	ErrPackageVCSRequired            = errors.New("package vcs required")
	ErrPackageRepoRootRequired       = errors.New("package repository root required")
	ErrPackageRepoRootInvalid        = errors.New("package repository root invalid")
	ErrPackageRepoRootSchemeRequired = errors.New("package repository root scheme required")
	ErrPackageRepoRootSchemeInvalid  = errors.New("package repository root scheme invalid")
	ErrPackageRepoRootHostInvalid    = errors.New("package repository root host invalid")
	ErrPackageRefChangeRejected      = errors.New("Package Reference Change Rejected")
	ErrChangelogRecordNotFound       = errors.New("changelog record not found")
)
