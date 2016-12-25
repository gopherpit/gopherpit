// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package certificate

import "time"

// Service defines functions that Certificate provider must have.
type Service interface {
	ManagementService
	ACMEService
	ACMEUserService
}

// Getter provides interface to get single certificate. It is most useful
// for services that are only consumers of certificates.
type Getter interface {
	// Certificate returns a Certificate for provided FQDN.
	Certificate(fqdn string) (c *Certificate, err error)
}

// ManagementService defines most basic functionality for certificate management.
type ManagementService interface {
	Getter
	// UpdateCertificate alters the fields of existing Certificate.
	UpdateCertificate(fqdn string, o *Options) (c *Certificate, err error)
	// DeleteCertificate deletes an existing Certificate for a
	// provided FQDN and returns it.
	DeleteCertificate(fqdn string) (c *Certificate, err error)
	// Certificates retrieves a paginated list of Certificate instances
	// ordered by FQDN.
	Certificates(start string, limit int) (page *CertificatesPage, err error)
	// CertificatesInfoByExpiry retrieves a paginated list of Info instances
	// ordered by expiration time.
	CertificatesInfoByExpiry(since time.Time, start string, limit int) (page *InfosPage, err error)
}

// ACMEService defines functionality required to obtain
// SSL/TLS certificate from ACME provider.
type ACMEService interface {
	// ObtainCertificate requests a new SSL/TLS certificate from
	// ACME provider and returns an instance of Certificate.
	ObtainCertificate(fqdn string) (c *Certificate, err error)
	// IsCertificateBeingObtained tests if certificate is being obtained currently.
	// It can be used as a locking mechanism.
	IsCertificateBeingObtained(fqdn string) (yes bool, err error)
	// ACMEChallenge returns an instance of ACMEChallenge for a FQDN.
	ACMEChallenge(fqdn string) (c *ACMEChallenge, err error)
	// UpdateACMEChallenge alters the fields of existing ACMEChallenge.
	UpdateACMEChallenge(fqdn string, o *ACMEChallengeOptions) (c *ACMEChallenge, err error)
	// DeleteACMEChallenge deletes an existing ACMEChallenge for a
	// provided FQDN and returns it.
	DeleteACMEChallenge(fqdn string) (c *ACMEChallenge, err error)
	// ACMEChallenges retrieves a paginated list of ACMEChallenge instances.
	ACMEChallenges(start string, limit int) (page *ACMEChallengesPage, err error)
}

// ACMEUserService handlers ACME user.
type ACMEUserService interface {
	// ACMEUser returns ACME user with ACME authentication details.
	ACMEUser() (u *ACMEUser, err error)
	// RegisterACMEUser registers and saves ACME user authentication data.
	RegisterACMEUser(directoryURL, email string) (u *ACMEUser, err error)
}

// Certificate holds data related to SSL/TLS certificate.
type Certificate struct {
	FQDN           string     `json:"fqdn"`
	ExpirationTime *time.Time `json:"expiration-time,omitempty"`
	Cert           string     `json:"cert,omitempty"`
	Key            string     `json:"key,omitempty"`
	ACMEURL        string     `json:"acme-url,omitempty"`
	ACMEURLStable  string     `json:"acme-url-stable,omitempty"`
	ACMEAccount    string     `json:"acme-account,omitempty"`
}

// Certificates is a list of Certificate instances.
type Certificates []Certificate

// CertificatesPage is a paginated list of Certificate instances.
type CertificatesPage struct {
	Certificates Certificates `json:"certificates"`
	Previous     string       `json:"previous,omitempty"`
	Next         string       `json:"next,omitempty"`
	Count        int          `json:"count,omitempty"`
}

// Options is a structure with parameters as pointers to set
// certificate data. If a parameter is nil, the corresponding
// Certificate parameter will not be changed.
type Options struct {
	Cert          *string `json:"cert,omitempty"`
	Key           *string `json:"key,omitempty"`
	ACMEURL       *string `json:"acme-url,omitempty"`
	ACMEURLStable *string `json:"acme-url-stable,omitempty"`
	ACMEAccount   *string `json:"acme-account,omitempty"`
}

// Info is a subset of Certificate structure fields to provide
// information about expiration time and ACME issuer.
type Info struct {
	FQDN           string     `json:"fqdn"`
	ExpirationTime *time.Time `json:"expiration-time,omitempty"`
	ACMEURL        string     `json:"acme-url,omitempty"`
	ACMEURLStable  string     `json:"acme-url-stable,omitempty"`
	ACMEAccount    string     `json:"acme-account,omitempty"`
}

// Infos is a list of Info instances.
type Infos []Info

// InfosPage is a paginated list of Info instances.
type InfosPage struct {
	Infos    Infos  `json:"infos"`
	Previous string `json:"previous,omitempty"`
	Next     string `json:"next,omitempty"`
	Count    int    `json:"count,omitempty"`
}

// ACMEUser is hods data about authentication to ACME provider.
type ACMEUser struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	PrivateKey   []byte `json:"private-key"`
	URL          string `json:"url"`
	NewAuthzURL  string `json:"new-authz-url"`
	DirectoryURL string `json:"directory-url"`
}

// ACMEChallenge provides data about ACME challenge for
// new certificate issue.
type ACMEChallenge struct {
	FQDN    string `json:"fqdn"`
	Token   string `json:"token,omitempty"`
	KeyAuth string `json:"key-auth,omitempty"`
}

// ACMEChallenges is a list of ACMEChallenge instances.
type ACMEChallenges []ACMEChallenge

// ACMEChallengesPage is a paginated list of ACMEChallenge instances.
type ACMEChallengesPage struct {
	ACMEChallenges ACMEChallenges `json:"acme-challenges"`
	Previous       string         `json:"previous,omitempty"`
	Next           string         `json:"next,omitempty"`
	Count          int            `json:"count,omitempty"`
}

// ACMEChallengeOptions is a structure with parameters as
// pointers to set ACME challenge data. If a parameter is nil,
// the corresponding ACMEChallenge parameter will not be changed.
type ACMEChallengeOptions struct {
	Token   *string `json:"token,omitempty"`
	KeyAuth *string `json:"key-auth,omitempty"`
}

// Errors that are related to the Certificate Service.
var (
	CertificateNotFound   = NewError(1000, "certificate not found")
	CertificateInvalid    = NewError(1001, "certificate invalid")
	FQDNMissing           = NewError(1100, "fqdn missing")
	FQDNInvalid           = NewError(1101, "fqdn invalid")
	FQDNExists            = NewError(1102, "fqdn exists")
	ACMEUserNotFound      = NewError(1200, "acme user not found")
	ACMEUserEmailInvalid  = NewError(1201, "acme user email invalid")
	ACMEChallengeNotFound = NewError(1300, "acme challenge not found")
)
