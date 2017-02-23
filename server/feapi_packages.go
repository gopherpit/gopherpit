// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
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
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/publicsuffix"
	"resenje.org/httputils"
	"resenje.org/jsonresponse"
	"resenje.org/marshal"

	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

var (
	fqdnRegex        = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	hostAndPortRegex = regexp.MustCompile(`^([a-z0-9]+[\-a-z0-9\.]*)(?:\:\d+)?$`)
	urlRegex         = regexp.MustCompile(`^((([A-Za-z]{3,9}:(?:\/\/)?)(?:[\-;:&=\+\$,\w]+@)?[A-Za-z0-9\.\-]+|(?:www\.|[\-;:&=\+\$,\w]+@)[A-Za-z0-9\.\-]+)((?:\/[\+~%\/\.\w\-_]*)?\??(?:[\-\+=&;%@\.\w_]*)#?(?:[\.\!\/\\\w]*))?)$`)
)

func (s Server) certificateFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := s.PackagesService.Domain(id)
	switch err {
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case nil:
	default:
		s.logger.Errorf("certificate fe api: domain %s: %s", id, err)
		jsonServerError(w, err)
		return
	}

	if domain.OwnerUserID != u.ID {
		jsonresponse.Forbidden(w, nil)
		return
	}

	if domain.CertificateIgnoreMissing {
		defer func() {
			False := false
			for {
				if _, err = s.PackagesService.UpdateDomain(domain.ID, &packages.DomainOptions{
					CertificateIgnoreMissing: &False,
				}, u.ID); err != nil {
					s.logger.Errorf("certificate fe api: update domain %s: certificate ignore missing false: %s", domain.FQDN, err)
					time.Sleep(60 * time.Second)
					continue
				}
				return
			}
		}()
	}

	certificate, err := s.CertificateService.ObtainCertificate(domain.FQDN)
	if err != nil {
		s.logger.Warningf("certificate fe api: obtain certificate: %s: %s", domain.FQDN, err)
		jsonresponse.BadRequest(w, httputils.NewError("Unable to obtain TLS certificate."))
		return
	}
	s.logger.Infof("certificate api: obtain certificate: success for %s: expiration time: %s", certificate.FQDN, certificate.ExpirationTime)

	s.auditf(r, nil, "obtain certificate", "%s: %s", certificate.FQDN, certificate.ExpirationTime)

	jsonresponse.OK(w, nil)
}

type domainFEAPIRequest struct {
	FQDN              string           `json:"fqdn"`
	CertificateIgnore marshal.Checkbox `json:"certificate-ignore"`
	Disabled          marshal.Checkbox `json:"disabled"`
}

type domainSecret struct {
	Domain string `json:"domain"`
	Secret string `json:"secret"`
}

type validationFormErrorResponse struct {
	httputils.FormErrors
	Secrets []domainSecret `json:"secrets"`
}

func (s Server) domainFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	request := domainFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("domain fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, httputils.NewError("Invalid data."))
		return
	}

	fqdn := strings.TrimSpace(request.FQDN)
	if fqdn == "" {
		s.logger.Warning("domain fe api: request: fqdn empty")
		jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Fully qualified domain name is required."))
		return
	}

	if !fqdnRegex.MatchString(fqdn) && fqdn != s.Domain {
		jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Fully qualified domain name is invalid."))
		return
	}

	var domain *packages.Domain
	if id != "" {
		domain, err = s.PackagesService.Domain(id)
		if err != nil {
			switch err {
			case packages.DomainNotFound:
				jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
				return
			case nil:
			default:
				s.logger.Errorf("domain fe api: domain %s: %s", id, err)
				jsonServerError(w, err)
				return
			}
		}
	}

	if (domain == nil || domain.FQDN != fqdn) && s.Domain != "" {
		switch {
		case fqdn == s.Domain, strings.HasSuffix(fqdn, "."+s.Domain):
			if strings.Count(fqdn, ".") > strings.Count(s.Domain, ".")+1 {
				jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", fmt.Sprintf("Only one subdomain is allowed for domain %s", s.Domain)))
				return
			}
		case !s.SkipDomainVerification:
			publicSuffix, icann := publicsuffix.PublicSuffix(fqdn)
			if !icann {
				jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", fmt.Sprintf("Top level domain %s is not an ICANN domain.", publicSuffix)))
				return
			}
			if fqdn == publicSuffix {
				jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", fmt.Sprintf("The domain %s is an ICANN domain.", publicSuffix)))
				return
			}

			secrets := []domainSecret{}
			domainParts := strings.Split(fqdn, ".")
			startIndex := len(domainParts) - strings.Count(publicSuffix, ".") - 2
			if startIndex < 0 {
				jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Fully qualified domain name is invalid."))
				return
			}

			domain, err = s.PackagesService.Domain(fqdn)
			if err != nil {
				if err != packages.DomainNotFound {
					s.logger.Errorf("domain fe api: domain %s: %s", fqdn, err)
					jsonServerError(w, err)
					return
				}
			} else {
				jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Domain already exists."))
				return
			}

			d := publicSuffix
			var secret, verificationDomain string
			var verified bool
			var x [20]byte
			for i := startIndex; i >= 0; i-- {
				d = fmt.Sprintf("%s.%s", domainParts[i], d)

				x = sha1.Sum(append(s.salt, []byte(u.ID+d)...))
				secret = base64.URLEncoding.EncodeToString(x[:])

				verificationDomain = s.VerificationSubdomain + "." + d

				verified, err = verifyDomain(verificationDomain, secret)
				if err != nil {
					s.logger.Errorf("domain fe api: verify domain: %s: %s", verificationDomain, err)
				}
				if verified {
					break
				}

				secrets = append(secrets, domainSecret{
					Domain: verificationDomain,
					Secret: secret,
				})
			}

			if !verified {
				jsonresponse.BadRequest(w, validationFormErrorResponse{
					FormErrors: httputils.NewFieldError("fqdn", "Domain is not verified."),
					Secrets:    secrets,
				})
				return
			}
		}
	}

	for _, d := range s.ForbiddenDomains {
		if d == fqdn {
			jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Domain is not available"))
			return
		}
	}

	disabled := request.Disabled.Bool()

	True := true

	var editedDomain *packages.Domain
	if id == "" {
		editedDomain, err = s.PackagesService.AddDomain(&packages.DomainOptions{
			FQDN:        &request.FQDN,
			OwnerUserID: &u.ID,
			Disabled:    &disabled,

			CertificateIgnoreMissing: &True,
		}, u.ID)
	} else {
		certificateIgnore := request.CertificateIgnore.Bool()
		editedDomain, err = s.PackagesService.UpdateDomain(id, &packages.DomainOptions{
			FQDN:              &request.FQDN,
			CertificateIgnore: &certificateIgnore,
			Disabled:          &disabled,
		}, u.ID)
	}
	if err != nil {
		switch err {
		case packages.DomainFQDNRequired:
			jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Domain fully qualified domain name is required."))
			return
		case packages.DomainOwnerUserIDRequired:
			jsonresponse.BadRequest(w, httputils.NewError("Domain user is required."))
			return
		case packages.DomainNotFound:
			jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Unknown domain."))
			return
		case packages.DomainAlreadyExists:
			jsonresponse.BadRequest(w, httputils.NewFieldError("fqdn", "Domain is already registered."))
			return
		case nil:
		default:
			s.logger.Errorf("domain fe api: add/update domain %s: %s", id, err)
			jsonServerError(w, err)
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
						s.logger.Errorf("domain fe api: update domain %s: certificate ignore missing false: %s", editedDomain.FQDN, err)
						time.Sleep(60 * time.Second)
						continue
					}
					return
				}
			}()
			certificate, err := s.CertificateService.ObtainCertificate(editedDomain.FQDN)
			if err != nil {
				s.logger.Errorf("domain fe api: obtain certificate: %s: %s", editedDomain.FQDN, err)
				return
			}
			s.logger.Infof("domain fe api: obtain certificate: success for %s: expiration time: %s", certificate.FQDN, certificate.ExpirationTime)
		}()
	}

	action := "domain update"
	if id == "" {
		action = "domain add"
	}
	s.auditf(r, request, action, "%s: %s", editedDomain.ID, editedDomain.FQDN)

	jsonresponse.OK(w, editedDomain)
}

func (s Server) domainDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	deletedDomain, err := s.PackagesService.DeleteDomain(id, u.ID)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
			return
		case nil:
		default:
			s.logger.Errorf("domain delete fe api: domain %s: %s", id, err)
			jsonServerError(w, err)
			return
		}
	}

	s.logger.Debugf("domain delete fe api: %s deleted by %s", deletedDomain.ID, u.ID)

	s.auditf(r, nil, "domain delete", "%s: %s", deletedDomain.ID, deletedDomain.FQDN)

	jsonresponse.OK(w, deletedDomain)
}

type userIDRequest struct {
	ID string `json:"id"`
}

func (s Server) domainUserGrantFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	request := userIDRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("domain user grant fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, httputils.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		jsonresponse.BadRequest(w, httputils.NewFieldError("id", "User is required."))
		return
	}

	domainID := mux.Vars(r)["id"]

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, httputils.NewError("You are already granted."))
		return
	}

	grantUser, err := s.UserService.User(request.ID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain user grant fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, httputils.NewError("Unknown user."))
			return
		}
		s.logger.Errorf("domain user grant fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	err = s.PackagesService.AddUserToDomain(domainID, grantUser.ID, u.ID)
	switch err {
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case packages.UserExists:
		jsonresponse.BadRequest(w, httputils.NewError("This user is already granted."))
		return
	case packages.Forbidden:
		jsonresponse.BadRequest(w, httputils.NewError("You do not have permission to revoke user."))
		return
	case nil:
	default:
		s.logger.Errorf("domain user grant fe api: add user to domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	s.audit(r, grantUser.ID, "domain user grant", domainID)

	jsonresponse.OK(w, nil)
}

func (s Server) domainUserRevokeFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	request := userIDRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("domain user grant fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, httputils.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		jsonresponse.BadRequest(w, httputils.NewError("User is required."))
		return
	}

	domainID := mux.Vars(r)["id"]

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, httputils.NewError("You can not revoke yourself."))
		return
	}

	revokeUser, err := s.UserService.User(request.ID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain user revoke fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, httputils.NewError("Unknown user."))
			return
		}
		s.logger.Errorf("domain user revoke fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	err = s.PackagesService.RemoveUserFromDomain(domainID, revokeUser.ID, u.ID)
	switch err {
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case packages.UserDoesNotExist:
		jsonresponse.BadRequest(w, httputils.NewError("This user is not granted."))
		return
	case packages.Forbidden:
		jsonresponse.BadRequest(w, httputils.NewError("You do not have permission to revoke user."))
		return
	case nil:
	default:
		s.logger.Errorf("domain user revoke fe api: revoke user form domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	s.audit(r, revokeUser.ID, "domain user revoke", domainID)

	jsonresponse.OK(w, nil)
}

type domainOwnerChangeFEAPIRequest struct {
	ID string `json:"id"`
}

func (s Server) domainOwnerChangeFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	domainID := mux.Vars(r)["id"]

	request := domainOwnerChangeFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("domain owner change fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, httputils.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		s.logger.Warning("domain owner change fe api: request: id empty")
		jsonresponse.BadRequest(w, httputils.NewError("User ID is required."))
		return
	}

	domain, err := s.PackagesService.Domain(domainID)
	if err != nil {
		switch err {
		case packages.DomainNotFound:
			jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
			return
		case nil:
		default:
			s.logger.Errorf("domain fe api: domain %s: %s", domainID, err)
			jsonServerError(w, err)
			return
		}
	}

	if domain.OwnerUserID != u.ID {
		jsonresponse.BadRequest(w, httputils.NewError("You do not have permission to change the owner of this domain."))
		return
	}

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, httputils.NewError("You are already the owner."))
		return
	}

	owner, err := s.UserService.User(request.ID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain owner change fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, httputils.NewError("Unknown user."))
			return
		}
		s.logger.Errorf("domain user revoke fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	domain, err = s.PackagesService.UpdateDomain(domainID, &packages.DomainOptions{
		OwnerUserID: &owner.ID,
	}, u.ID)
	switch err {
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case nil:
	default:
		s.logger.Errorf("domain owner change fe api: update domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	s.auditf(r, request, "domain owner change", "%s: %s to %s: %s", domain.ID, domain.FQDN, owner.ID, owner.Email)

	jsonresponse.OK(w, domain)
}

type packageFEAPIRequest struct {
	DomainID    string           `json:"domainId"`
	Path        string           `json:"path"`
	VCS         packages.VCS     `json:"vcs"`
	RepoRoot    string           `json:"repoRoot"`
	RefType     string           `json:"refType"`
	RefName     string           `json:"refName"`
	GoSource    string           `json:"goSource"`
	RedirectURL string           `json:"redirectUrl"`
	Disabled    marshal.Checkbox `json:"disabled"`
}

var vcsSchemas = map[packages.VCS][]string{
	packages.VCSGit:        {"https", "http", "git", "git+ssh", "ssh"},
	packages.VCSMercurial:  {"https", "http", "ssh"},
	packages.VCSBazaar:     {"https", "http", "bzr", "bzr+ssh"},
	packages.VCSSubversion: {"https", "http", "svn", "svn+ssh"},
}

func (s Server) packageFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	request := packageFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Warningf("package fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, httputils.NewError("Invalid data."))
		return
	}

	errors := httputils.FormErrors{}

	request.DomainID = strings.TrimSpace(request.DomainID)
	if request.DomainID == "" {
		errors.AddFieldError("domainId", "Domain is required.")
	}

	request.Path = strings.TrimSpace(request.Path)
	if request.Path == "" {
		errors.AddFieldError("path", "Path is required.")
	}

	if request.VCS == "" {
		errors.AddFieldError("vcs", "VCS is required.")
	}

	var repoRoot *url.URL
	request.RepoRoot = strings.TrimSpace(request.RepoRoot)
	if request.RepoRoot == "" {
		errors.AddFieldError("repoRoot", "Repository is required.")
	} else {
		var err error
		repoRoot, err = url.Parse(request.RepoRoot)
		switch {
		case err != nil:
			s.logger.Warningf("package fe api: %s %s: invalid repository url: %s: %s", request.DomainID, request.Path, request.RepoRoot, err)
			errors.AddFieldError("repoRoot", "Invalid Repository URL.")
		case request.VCS != "":
			if repoRoot.Scheme == "" {
				errors.AddFieldError("repoRoot", fmt.Sprintf("Repository URL requires a URL scheme (%s).", strings.Join(vcsSchemas[request.VCS], ", ")))
				break
			}
			ok := false
			for _, s := range vcsSchemas[request.VCS] {
				if repoRoot.Scheme == s {
					ok = true
					break
				}
			}
			if !ok {
				vcs := string(request.VCS)
				for _, i := range vcsInfos {
					if i.VCS == request.VCS {
						vcs = i.Name
						break
					}
				}
				errors.AddFieldError("repoRoot", fmt.Sprintf("Invalid scheme \"%s\". For %s repository it should be one of (%s).", repoRoot.Scheme, vcs, strings.Join(vcsSchemas[request.VCS], ", ")))
			}
			if !hostAndPortRegex.MatchString(repoRoot.Host) {
				errors.AddFieldError("repoRoot", fmt.Sprintf("Invalid domain and port \"%s\".", repoRoot.Host))
			}
		}
	}

	switch request.RefType {
	case "", "tag", "branch":
	default:
		errors.AddFieldError("refType", "Reference type must be branch or tag.")
	}

	if request.RefType != "" && request.RefName == "" {
		errors.AddFieldError("refName", "Reference name is required if reference type is selected.")
	}

	if request.RefName != "" && (request.VCS != packages.VCSGit || (request.VCS == packages.VCSGit && repoRoot != nil && !(repoRoot.Scheme == "http" || repoRoot.Scheme == "https"))) {
		errors.AddFieldError("refName", "Reference change is allowed only for Git HTTP and HTTPS repositeries.")
	}

	if request.RedirectURL != "" && !urlRegex.MatchString(request.RedirectURL) {
		errors.AddFieldError("redirectUrl", "Invalid URL.")
	}

	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	if !strings.HasPrefix(request.Path, "/") {
		request.Path = "/" + request.Path
	}

	disabled := request.Disabled.Bool()

	var p *packages.Package
	if id == "" {
		p, err = s.PackagesService.AddPackage(&packages.PackageOptions{
			Domain:      &request.DomainID,
			Path:        &request.Path,
			VCS:         &request.VCS,
			RepoRoot:    &request.RepoRoot,
			RefType:     &request.RefType,
			RefName:     &request.RefName,
			GoSource:    &request.GoSource,
			RedirectURL: &request.RedirectURL,
			Disabled:    &disabled,
		}, u.ID)
	} else {
		p, err = s.PackagesService.UpdatePackage(id, &packages.PackageOptions{
			Domain:      &request.DomainID,
			Path:        &request.Path,
			VCS:         &request.VCS,
			RepoRoot:    &request.RepoRoot,
			RefType:     &request.RefType,
			RefName:     &request.RefName,
			GoSource:    &request.GoSource,
			RedirectURL: &request.RedirectURL,
			Disabled:    &disabled,
		}, u.ID)
	}
	switch err {
	case packages.Forbidden:
		jsonresponse.BadRequest(w, httputils.NewError("You do not have permission to add packages to this domain."))
		return
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case packages.PackageNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown package."))
		return
	case packages.PackageDomainRequired:
		jsonresponse.BadRequest(w, httputils.NewError("Domain is required."))
		return
	case packages.PackagePathRequired:
		jsonresponse.BadRequest(w, httputils.NewError("Path is required."))
		return
	case packages.PackageVCSRequired:
		jsonresponse.BadRequest(w, httputils.NewError("VCS is required."))
		return
	case packages.PackageRepoRootRequired:
		jsonresponse.BadRequest(w, httputils.NewError("Repository is required."))
		return
	case packages.PackageAlreadyExists:
		jsonresponse.BadRequest(w, httputils.NewError("Package already exists."))
		return
	case nil:
	default:
		s.logger.Errorf("package fe api: add/update package %s: %s", id, err)
		jsonServerError(w, err)
		return
	}

	action := "package update"
	if id == "" {
		action = "package add"
	}
	s.auditf(r, request, action, "%s %s (domain: %s)", p.ID, p.ImportPrefix(), p.Domain.ID)

	jsonresponse.OK(w, p)
}

func (s Server) packageDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	// Delete package checks permissions.
	p, err := s.PackagesService.DeletePackage(id, u.ID)
	switch err {
	case packages.Forbidden:
		jsonresponse.BadRequest(w, httputils.NewError("You do not have permission to add packages to this domain."))
		return
	case packages.DomainNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown domain."))
		return
	case packages.PackageNotFound:
		jsonresponse.BadRequest(w, httputils.NewError("Unknown package."))
		return
	case nil:
	default:
		s.logger.Errorf("package delete fe api: delete package %s: %s", id, err)
		jsonServerError(w, err)
		return
	}

	s.logger.Debugf("package delete fe api: %s deleted by %s", p.ID, u.ID)

	s.auditf(r, nil, "package delete", "%s: %s", p.ID, p.ImportPrefix)

	jsonresponse.OK(w, p)
}
