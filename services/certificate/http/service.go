// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpCertificate provides a Service that is a HTTP client to
// an external certificate service that can respond to HTTP requests defined here.
package httpCertificate

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/certificate"
)

// Service implements gopherpit.com/gopherpit/services/certificates.Service interface.
type Service struct {
	Client *apiClient.Client
}

// NewService creates a new Service and injects certificate.ErrorRegistry
// in the API Client.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = certificate.ErrorRegistry
	return &Service{Client: c}
}

// Certificate retrieves an existing Certificate instance by making a HTTP GET request
// to {Client.Endpoint}/certificates/{fqdn}.
func (s Service) Certificate(fqdn string) (c *certificate.Certificate, err error) {
	err = s.Client.JSON("GET", "/certificates/"+fqdn, nil, nil, c)
	return
}

// ObtainCertificateRequest is a structure that is passed as JSON-encoded body
// to ObtainCertificate HTTP request.
type ObtainCertificateRequest struct {
	FQDN string `json:"fqdn"`
}

// ObtainCertificate obtains a new certificate from ACME provider by making a HTTP POST
// request to {Client.Endpoint}/certificates. Post body is a JSON-encoded
// ObtainCertificateRequest instance.
func (s Service) ObtainCertificate(fqdn string) (c *certificate.Certificate, err error) {
	body, err := json.Marshal(ObtainCertificateRequest{
		FQDN: fqdn,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/certificates", nil, bytes.NewReader(body), c)
	return
}

// IsCertificateBeingObtainedResponse is expected structure of JSON-encoded response
// body for IsCertificateBeingObtained HTTP request.
type IsCertificateBeingObtainedResponse struct {
	Yes bool `json:"yes"`
}

// IsCertificateBeingObtained tests if certificate is being obtained currently
// by making a HTTP GET request to {Client.Endpoint}/certificates/{fqdn}/being-obtained.
// Expected response body is a JSON-encoded instance of IsCertificateBeingObtainedResponse.
func (s Service) IsCertificateBeingObtained(fqdn string) (yes bool, err error) {
	response := &IsCertificateBeingObtainedResponse{}
	err = s.Client.JSON("GET", "/certificates/"+fqdn+"/being-obtained", nil, nil, response)
	yes = response.Yes
	return
}

// UpdateCertificate changes the data of an existing Certificate by making a HTTP POST
// request to {Client.Endpoint}/certificates/{fqdn}. Post body is a JSON-encoded
// certificate.Options instance.
func (s Service) UpdateCertificate(fqdn string, o *certificate.Options) (c *certificate.Certificate, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/certificates/"+fqdn, nil, bytes.NewReader(body), c)
	return
}

// DeleteCertificate deletes an existing Certificate by making a HTTP DELETE request
// to {Client.Endpoint}/certificates/{fqdn}.
func (s Service) DeleteCertificate(fqdn string) (c *certificate.Certificate, err error) {
	err = s.Client.JSON("DELETE", "/certificates/"+fqdn, nil, nil, c)
	return
}

// Certificates retrieves a paginated list of Certificate instances
// ordered by FQDN, by making a HTTP GET request to {Client.Endpoint}/certificates.
func (s Service) Certificates(start string, limit int) (page *certificate.CertificatesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/certificates", query, nil, page)
	return
}

// CertificatesInfoByExpiry retrieves a paginated list of Info instances
// ordered by expiration time by making a HTTP GET request to
// {Client.Endpoint}/certificates-info-by-expiry.
func (s Service) CertificatesInfoByExpiry(since time.Time, start string, limit int) (page *certificate.InfosPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if !since.IsZero() {
		query.Set("since", since.String())
	}
	err = s.Client.JSON("GET", "/certificates-info-by-expiry", query, nil, page)
	return
}

// ACMEUser returns ACME user with ACME authentication details by making a HTTP GET
// request to {Client.Endpoint}/acme/user.
func (s Service) ACMEUser() (u *certificate.ACMEUser, err error) {
	err = s.Client.JSON("GET", "/acme/user", nil, nil, u)
	return
}

// RegisterACMEUserRequest is a structure that is passed as JSON-encoded body
// to RegisterACMEUser HTTP request.
type RegisterACMEUserRequest struct {
	DirectoryURL string `json:"directory-url"`
	Email        string `json:"email"`
}

// RegisterACMEUser registers and saves ACME user authentication data by making a
// HTTP POST request to {Client.Endpoint}/acme/user. Post body is a JSON-encoded
// RegisterACMEUserRequest instance.
func (s Service) RegisterACMEUser(directoryURL, email string) (u *certificate.ACMEUser, err error) {
	body, err := json.Marshal(RegisterACMEUserRequest{
		DirectoryURL: directoryURL,
		Email:        email,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/acme/user", nil, bytes.NewReader(body), u)
	return
}

// ACMEChallenge returns an instance of ACMEChallenge for a FQDN by making a
// HTTP GET request to {Client.Endpoint}/acme/challenges/{fqdn}.
func (s Service) ACMEChallenge(fqdn string) (c *certificate.ACMEChallenge, err error) {
	err = s.Client.JSON("GET", "/acme/challenges/"+fqdn, nil, nil, c)
	return
}

// UpdateACMEChallenge alters the fields of existing ACMEChallenge by making a
// HTTP POST request to {Client.Endpoint}/acme/challenges/{fqdn}. Post body is a JSON-encoded
// certificate.ACMEChallengeOptions instance.
func (s Service) UpdateACMEChallenge(fqdn string, o *certificate.ACMEChallengeOptions) (c *certificate.ACMEChallenge, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/acme/challenges/"+fqdn, nil, bytes.NewReader(body), c)
	return
}

// DeleteACMEChallenge deletes an existing ACMEChallenge for a provided FQDN
// and returns it by making a HTTP DELETE request to {Client.Endpoint}/acme/challenges/{fqdn}.
func (s Service) DeleteACMEChallenge(fqdn string) (c *certificate.ACMEChallenge, err error) {
	err = s.Client.JSON("DELETE", "/acme/challenges/"+fqdn, nil, nil, c)
	return
}

// ACMEChallenges retrieves a paginated list of ACMEChallenge instances by making a
// HTTP GET request to {Client.Endpoint}/acme/challenges.
func (s Service) ACMEChallenges(start string, limit int) (page *certificate.ACMEChallengesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/acme/challenges", query, nil, page)
	return
}
