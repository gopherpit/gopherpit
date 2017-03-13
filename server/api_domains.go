// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/gorilla/mux"
	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

func (s Server) domainAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			s.logger.Warningf("domain api: domain %s: %s", id, err)
			jsonresponse.NotFound(w, api.ErrDomainNotFound)
			return
		case nil:
		default:
			s.logger.Errorf("domain api: domain %s: %s", id, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
	}

	found := domain.OwnerUserID == u.ID
	if !found {
		domainUsers, err := s.PackagesService.DomainUsers(id)
		if err != nil {
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain api: domain users %s: %s", id, err)
				jsonresponse.NotFound(w, api.ErrDomainNotFound)
				return
			}
			s.logger.Errorf("domain api: domain users %s: %s", id, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
		for _, uid := range domainUsers.UserIDs {
			if u.ID == uid {
				found = true
				break
			}
		}
	}

	if !found {
		s.logger.Errorf("domain api: domain %s: does not belong to user %s", id, u.ID)
		jsonresponse.Forbidden(w, nil)
		return
	}

	jsonresponse.OK(w, packagesDomainToAPIDomain(*domain))
}

func (s Server) domainTokensAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	fqdn := vars["id"]

	publicSuffix, icann := publicsuffix.PublicSuffix(fqdn)
	if !icann {
		s.logger.Warningf("domain token api: %q: user %s: domain not icann", fqdn, u.ID)
		jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
		return
	}
	if fqdn == publicSuffix {
		s.logger.Warningf("domain token api: %q: user %s: domain is public suffix", fqdn, u.ID)
		jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
		return
	}

	tokens := []api.DomainToken{}
	domainParts := strings.Split(fqdn, ".")
	startIndex := len(domainParts) - strings.Count(publicSuffix, ".") - 2
	if startIndex < 0 {
		s.logger.Warningf("domain token api: %q: user %s: domain is invalid", fqdn, u.ID)
		jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
		return
	}

	d := publicSuffix
	var token, verificationDomain string
	var x [20]byte
	for i := startIndex; i >= 0; i-- {
		d = fmt.Sprintf("%s.%s", domainParts[i], d)

		x = sha1.Sum(append(s.salt, []byte(u.ID+d)...))
		token = base64.URLEncoding.EncodeToString(x[:])

		verificationDomain = s.VerificationSubdomain + "." + d

		tokens = append(tokens, api.DomainToken{
			Domain: verificationDomain,
			Token:  token,
		})
	}

	jsonresponse.OK(w, api.DomainTokens{
		Tokens: tokens,
	})
}

func (s Server) updateDomainAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	warningf := func(format string, a ...interface{}) {
		s.logger.Warningf("update domain api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}
	errorf := func(format string, a ...interface{}) {
		s.logger.Errorf("update domain api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}

	request := api.DomainOptions{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		warningf("request decode: %s", err)
		jsonresponse.BadRequest(w, nil)
		return
	}

	if request.FQDN == nil {
		warningf("request: fqdn absent")
		jsonresponse.BadRequest(w, api.ErrDomainFQDNRequired)
		return
	}

	fqdn := strings.TrimSpace(*request.FQDN)
	if fqdn == "" {
		warningf("request: fqdn empty")
		jsonresponse.BadRequest(w, api.ErrDomainFQDNRequired)
		return
	}

	if !fqdnRegex.MatchString(fqdn) && fqdn != s.Domain {
		warningf("request: fqdn invalid")
		jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
		return
	}

	var domain *packages.Domain
	if id != "" {
		domain, err = s.PackagesService.Domain(id)
		if err != nil {
			switch err {
			case packages.DomainNotFound:
				warningf("get domain: %s", err)
				jsonresponse.BadRequest(w, api.ErrDomainNotFound)
				return
			case nil:
			default:
				errorf("get domain: %s", err)
				jsonresponse.InternalServerError(w, nil)
				return
			}
		}
	}

	skipDomainVerification := s.SkipDomainVerification
	if !skipDomainVerification {
		for _, d := range s.TrustedDomains {
			if fqdn == d || strings.HasSuffix(fqdn, "."+d) {
				skipDomainVerification = true
				break
			}
		}
	}

	for _, d := range s.ForbiddenDomains {
		if d == fqdn || strings.HasSuffix(fqdn, "."+d) {
			warningf("domain not available: %s", fqdn)
			jsonresponse.BadRequest(w, api.ErrDomainNotAvailable)
			return
		}
	}

	// New or changed domain fqdn verification.
	if (domain == nil || domain.FQDN != fqdn) && s.Domain != "" && !skipDomainVerification {
		switch {
		case fqdn == s.Domain, strings.HasSuffix(fqdn, "."+s.Domain):
			if strings.Count(fqdn, ".") > strings.Count(s.Domain, ".")+1 {
				warningf("domain with too many subdomains: %s", fqdn)
				jsonresponse.BadRequest(w, api.ErrDomainWithTooManySubdomains)
				return
			}
		default:
			publicSuffix, icann := publicsuffix.PublicSuffix(fqdn)
			if !icann {
				warningf("domain not icann: %s", fqdn)
				jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
				return
			}
			if fqdn == publicSuffix {
				warningf("domain is public suffix: %s", fqdn)
				jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
				return
			}

			domainParts := strings.Split(fqdn, ".")
			startIndex := len(domainParts) - strings.Count(publicSuffix, ".") - 2
			if startIndex < 0 {
				warningf("domain is invalid: %s", fqdn)
				jsonresponse.BadRequest(w, api.ErrDomainFQDNInvalid)
				return
			}

			domain, err = s.PackagesService.Domain(fqdn)
			if err != nil {
				if err != packages.DomainNotFound {
					errorf("get domain: %s: %s", fqdn, err)
					jsonresponse.InternalServerError(w, nil)
					return
				}
			} else {
				warningf("get domain: %s: %s", fqdn, err)
				jsonresponse.BadRequest(w, api.ErrDomainAlreadyExists)
				return
			}

			d := publicSuffix
			var token, verificationDomain string
			var verified bool
			var x [20]byte
			for i := startIndex; i >= 0; i-- {
				d = fmt.Sprintf("%s.%s", domainParts[i], d)

				x = sha1.Sum(append(s.salt, []byte(u.ID+d)...))
				token = base64.URLEncoding.EncodeToString(x[:])

				verificationDomain = s.VerificationSubdomain + "." + d

				verified, err = verifyDomain(verificationDomain, token)
				if err != nil {
					warningf("verify domain: %s: %s", verificationDomain, err)
				}
				if verified {
					break
				}
			}

			if !verified {
				warningf("domain needs verification: %s", fqdn)
				jsonresponse.BadRequest(w, api.ErrDomainNeedsVerification)
				return
			}
		}
	}

	var editedDomain *packages.Domain
	if id == "" {
		t := true
		ownerUserID := &u.ID
		if request.OwnerUserID != nil {
			owner, err := s.UserService.User(*request.OwnerUserID)
			if err != nil {
				if err == user.UserNotFound {
					warningf("get owner user: %s: %s", *request.OwnerUserID, err)
					jsonresponse.BadRequest(w, api.ErrUserDoesNotExist)
					return
				}
				errorf("get owner user: %s: %s", *request.OwnerUserID, err)
				jsonresponse.InternalServerError(w, nil)
				return
			}
			ownerUserID = &owner.ID
		}
		editedDomain, err = s.PackagesService.AddDomain(&packages.DomainOptions{
			FQDN:        request.FQDN,
			OwnerUserID: ownerUserID,
			Disabled:    request.Disabled,

			CertificateIgnoreMissing: &t,
		}, u.ID)
	} else {
		editedDomain, err = s.PackagesService.UpdateDomain(id, &packages.DomainOptions{
			FQDN:              request.FQDN,
			OwnerUserID:       request.OwnerUserID,
			CertificateIgnore: request.CertificateIgnore,
			Disabled:          request.Disabled,
		}, u.ID)
	}
	if err != nil {
		switch err {
		case packages.DomainFQDNRequired:
			warningf("add/update domain: %s: %s", fqdn, err)
			jsonresponse.BadRequest(w, api.ErrDomainFQDNRequired)
			return
		case packages.DomainNotFound:
			warningf("add/update domain: %s: %s", fqdn, err)
			jsonresponse.BadRequest(w, api.ErrDomainNotFound)
			return
		case packages.DomainAlreadyExists:
			warningf("add/update domain: %s: %s", fqdn, err)
			jsonresponse.BadRequest(w, api.ErrDomainAlreadyExists)
			return
		case nil:
		default:
			errorf("add/update domain: %s: %s", fqdn, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
	}

	// Obtain certificate only if:
	// - it is the new domain (id == "")
	// - TLS server is active (s.TLSEnabled == true)
	// - the domain as actually created (editedDomain != nil)
	if id == "" && s.TLSEnabled && editedDomain != nil {
		go func() {
			defer s.RecoveryService.Recover()
			defer func() {
				f := false
				for {
					if _, err = s.PackagesService.UpdateDomain(editedDomain.ID, &packages.DomainOptions{
						CertificateIgnoreMissing: &f,
					}, u.ID); err != nil {
						errorf("update domain %s: certificate ignore missing false: %s", editedDomain.FQDN, err)
						time.Sleep(60 * time.Second)
						continue
					}
					return
				}
			}()
			certificate, err := s.CertificateService.ObtainCertificate(editedDomain.FQDN)
			if err != nil {
				errorf("obtain certificate: %s: %s", editedDomain.FQDN, err)
				return
			}
			s.logger.Infof("update domain api: %q: user %s: obtain certificate: success for %s: expiration time: %s", id, u.ID, certificate.FQDN, certificate.ExpirationTime)
		}()
	}

	action := "domain update"
	if id == "" {
		action = "domain add"
	}
	s.auditf(r, request, action, "%s: %s", editedDomain.ID, editedDomain.FQDN)

	jsonresponse.OK(w, packagesDomainToAPIDomain(*editedDomain))
}

func (s Server) deleteDomainAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := s.PackagesService.DeleteDomain(id, u.ID)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			s.logger.Warningf("delete domain api: delete domain %s: %s", id, err)
			jsonresponse.NotFound(w, api.ErrDomainNotFound)
			return
		case nil:
		default:
			s.logger.Errorf("delete domain api: delete domain %s: %s", id, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
	}

	jsonresponse.OK(w, packagesDomainToAPIDomain(*domain))
}

func (s Server) domainsAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	startRef := r.URL.Query().Get("start")

	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		var err error
		limit, err = strconv.Atoi(l)
		if err != nil || limit < 1 || limit > api.MaxLimit {
			limit = 0
			return
		}
	}

	domains, err := s.PackagesService.DomainsByUser(u.ID, startRef, limit)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			s.logger.Warningf("domains api: domains by user %s: start ref %q: %s", u.ID, startRef, err)
			jsonresponse.NotFound(w, api.ErrDomainNotFound)
			return
		case packages.UserDoesNotExist:
			s.logger.Warningf("domains api: domains by user %s: %s", u.ID, err)
			jsonresponse.NotFound(w, api.ErrUserDoesNotExist)
			return
		case nil:
		default:
			s.logger.Errorf("domains api: domains by user %s: start ref %q: %s", u.ID, startRef, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
	}

	response := api.DomainsPage{
		Domains:  api.Domains{},
		Previous: domains.Previous,
		Next:     domains.Next,
		Count:    domains.Count,
	}

	for _, d := range domains.Domains {
		response.Domains = append(response.Domains, packagesDomainToAPIDomain(d))
	}

	jsonresponse.OK(w, response)
}

func (s Server) domainUsersAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain users api: domain %s: %s", id, err)
			jsonresponse.NotFound(w, api.ErrDomainNotFound)
			return
		}
		s.logger.Errorf("domain users api: domain %s: %s", id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	if domain.OwnerUserID != u.ID {
		s.logger.Warningf("domain users api: domain %s: user %s: is not the owner", id, u.ID)
		jsonresponse.Forbidden(w, nil)
		return
	}

	users, err := s.PackagesService.DomainUsers(id)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			s.logger.Warningf("domain users api: domain users %s: %s", id, err)
			jsonresponse.NotFound(w, api.ErrDomainNotFound)
			return
		case nil:
		default:
			s.logger.Errorf("domain users api: domain users %s: %s", id, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
	}

	jsonresponse.OK(w, api.DomainUsers{
		OwnerUserID: users.OwnerUserID,
		UserIDs:     users.UserIDs,
	})
}

func (s Server) grantDomainUserAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	domainID := vars["id"]
	userID := vars["user-id"]

	grantUser, err := s.UserService.User(userID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain user grant api: user %s: domain %s: get user %s: %s", u.ID, domainID, userID, err)
			jsonresponse.BadRequest(w, api.ErrUserDoesNotExist)
			return
		}
		s.logger.Errorf("domain user grant api: user %s: domain %s: get user %s: %s", u.ID, domainID, userID, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}
	err = s.PackagesService.AddUserToDomain(domainID, grantUser.ID, u.ID)
	switch err {
	case packages.DomainNotFound:
		s.logger.Warningf("domain user grant api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.BadRequest(w, api.ErrDomainNotFound)
	case packages.UserExists:
		s.logger.Warningf("domain user grant api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.BadRequest(w, api.ErrUserAlreadyGranted)
	case packages.Forbidden:
		s.logger.Warningf("domain user grant api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.Forbidden(w, nil)
	case nil:
		jsonresponse.OK(w, nil)
	default:
		s.logger.Errorf("domain user grant api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.InternalServerError(w, nil)
	}
}

func (s Server) revokeDomainUserAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	domainID := vars["id"]
	userID := vars["user-id"]

	revokeUser, err := s.UserService.User(userID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain user revoke api: user %s: domain %s: get user %s: %s", u.ID, domainID, userID, err)
			jsonresponse.BadRequest(w, api.ErrUserDoesNotExist)
			return
		}
		s.logger.Errorf("domain user revoke api: user %s: domain %s: get user %s: %s", u.ID, domainID, userID, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}
	err = s.PackagesService.RemoveUserFromDomain(domainID, revokeUser.ID, u.ID)
	switch err {
	case packages.DomainNotFound:
		s.logger.Warningf("domain user revoke api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.BadRequest(w, api.ErrDomainNotFound)
	case packages.UserDoesNotExist:
		s.logger.Warningf("domain user revoke api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.BadRequest(w, api.ErrUserNotGranted)
	case packages.Forbidden:
		s.logger.Warningf("domain user revoke api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.Forbidden(w, nil)
	case nil:
		jsonresponse.OK(w, nil)
	default:
		s.logger.Errorf("domain user revoke api: user %s: add user %s to domain %s: %s", u.ID, userID, domainID, err)
		jsonresponse.InternalServerError(w, nil)
	}
}
