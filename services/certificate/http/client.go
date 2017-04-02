// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpCertificate provides a HTTP client to
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

// Client implements gopherpit.com/gopherpit/services/certificates.Service interface.
type Client struct {
	*apiClient.Client
}

// NewClient creates a new Client.
func NewClient(c *apiClient.Client) *Client {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = errorRegistry
	return &Client{Client: c}
}

// Certificate retrieves an existing Certificate instance by making a HTTP GET request
// to {Client.Endpoint}/certificates/{fqdn}.
func (c Client) Certificate(fqdn string) (crt *certificate.Certificate, err error) {
	crt = &certificate.Certificate{}
	err = c.JSON("GET", "/certificates/"+fqdn, nil, nil, crt)
	err = getServiceError(err)
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
func (c Client) ObtainCertificate(fqdn string) (crt *certificate.Certificate, err error) {
	body, err := json.Marshal(ObtainCertificateRequest{
		FQDN: fqdn,
	})
	if err != nil {
		return
	}
	crt = &certificate.Certificate{}
	err = c.JSON("POST", "/certificates", nil, bytes.NewReader(body), crt)
	err = getServiceError(err)
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
func (c Client) IsCertificateBeingObtained(fqdn string) (yes bool, err error) {
	response := &IsCertificateBeingObtainedResponse{}
	err = c.JSON("GET", "/certificates/"+fqdn+"/being-obtained", nil, nil, response)
	err = getServiceError(err)
	yes = response.Yes
	return
}

// UpdateCertificate changes the data of an existing Certificate by making a HTTP POST
// request to {Client.Endpoint}/certificates/{fqdn}. Post body is a JSON-encoded
// certificate.Options instance.
func (c Client) UpdateCertificate(fqdn string, o *certificate.Options) (crt *certificate.Certificate, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	crt = &certificate.Certificate{}
	err = c.JSON("POST", "/certificates/"+fqdn, nil, bytes.NewReader(body), crt)
	err = getServiceError(err)
	return
}

// DeleteCertificate deletes an existing Certificate by making a HTTP DELETE request
// to {Client.Endpoint}/certificates/{fqdn}.
func (c Client) DeleteCertificate(fqdn string) (crt *certificate.Certificate, err error) {
	crt = &certificate.Certificate{}
	err = c.JSON("DELETE", "/certificates/"+fqdn, nil, nil, crt)
	err = getServiceError(err)
	return
}

// Certificates retrieves a paginated list of Certificate instances
// ordered by FQDN, by making a HTTP GET request to {Client.Endpoint}/certificates.
func (c Client) Certificates(start string, limit int) (page *certificate.CertificatesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	page = &certificate.CertificatesPage{}
	err = c.JSON("GET", "/certificates", query, nil, page)
	err = getServiceError(err)
	return
}

// CertificatesInfoByExpiry retrieves a paginated list of Info instances
// ordered by expiration time by making a HTTP GET request to
// {Client.Endpoint}/certificates-info-by-expiry.
func (c Client) CertificatesInfoByExpiry(since time.Time, start string, limit int) (page *certificate.InfosPage, err error) {
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
	page = &certificate.InfosPage{}
	err = c.JSON("GET", "/certificates-info-by-expiry", query, nil, page)
	err = getServiceError(err)
	return
}

// ACMEUser returns ACME user with ACME authentication details by making a HTTP GET
// request to {Client.Endpoint}/acme/user.
func (c Client) ACMEUser() (u *certificate.ACMEUser, err error) {
	err = c.JSON("GET", "/acme/user", nil, nil, u)
	err = getServiceError(err)
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
func (c Client) RegisterACMEUser(directoryURL, email string) (u *certificate.ACMEUser, err error) {
	body, err := json.Marshal(RegisterACMEUserRequest{
		DirectoryURL: directoryURL,
		Email:        email,
	})
	if err != nil {
		return
	}
	u = &certificate.ACMEUser{}
	err = c.JSON("POST", "/acme/user", nil, bytes.NewReader(body), u)
	err = getServiceError(err)
	return
}

// ACMEChallenge returns an instance of ACMEChallenge for a FQDN by making a
// HTTP GET request to {Client.Endpoint}/acme/challenges/{fqdn}.
func (c Client) ACMEChallenge(fqdn string) (ac *certificate.ACMEChallenge, err error) {
	ac = &certificate.ACMEChallenge{}
	err = c.JSON("GET", "/acme/challenges/"+fqdn, nil, nil, ac)
	err = getServiceError(err)
	return
}

// UpdateACMEChallenge alters the fields of existing ACMEChallenge by making a
// HTTP POST request to {Client.Endpoint}/acme/challenges/{fqdn}. Post body is a JSON-encoded
// certificate.ACMEChallengeOptions instance.
func (c Client) UpdateACMEChallenge(fqdn string, o *certificate.ACMEChallengeOptions) (ac *certificate.ACMEChallenge, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	ac = &certificate.ACMEChallenge{}
	err = c.JSON("POST", "/acme/challenges/"+fqdn, nil, bytes.NewReader(body), ac)
	err = getServiceError(err)
	return
}

// DeleteACMEChallenge deletes an existing ACMEChallenge for a provided FQDN
// and returns it by making a HTTP DELETE request to {Client.Endpoint}/acme/challenges/{fqdn}.
func (c Client) DeleteACMEChallenge(fqdn string) (ac *certificate.ACMEChallenge, err error) {
	ac = &certificate.ACMEChallenge{}
	err = c.JSON("DELETE", "/acme/challenges/"+fqdn, nil, nil, ac)
	err = getServiceError(err)
	return
}

// ACMEChallenges retrieves a paginated list of ACMEChallenge instances by making a
// HTTP GET request to {Client.Endpoint}/acme/challenges.
func (c Client) ACMEChallenges(start string, limit int) (page *certificate.ACMEChallengesPage, err error) {
	query := url.Values{}
	if start != "" {
		query.Set("start", start)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	page = &certificate.ACMEChallengesPage{}
	err = c.JSON("GET", "/acme/challenges", query, nil, page)
	err = getServiceError(err)
	return
}
