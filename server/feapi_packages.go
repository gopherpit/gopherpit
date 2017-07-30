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
	"resenje.org/jsonresponse"
	"resenje.org/marshal"
	"resenje.org/web"

	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

var (
	fqdnRegex        = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	hostAndPortRegex = regexp.MustCompile(`^([a-z0-9]+[\-a-z0-9\.]*)(?:\:\d+)?$`)
	urlRegex         = regexp.MustCompile(`^((([A-Za-z]{3,9}:(?:\/\/)?)(?:[\-;:&=\+\$,\w]+@)?[A-Za-z0-9\.\-]+|(?:www\.|[\-;:&=\+\$,\w]+@)[A-Za-z0-9\.\-]+)((?:\/[\+~%\/\.\w\-_]*)?\??(?:[\-\+=&;%@\.\w_]*)#?(?:[\.\!\/\\\w]*))?)$`)
)

func certificateFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := srv.PackagesService.Domain(id)
	switch err {
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case nil:
	default:
		srv.Logger.Errorf("certificate fe api: domain %s: %s", id, err)
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
				if _, err = srv.PackagesService.UpdateDomain(domain.ID, &packages.DomainOptions{
					CertificateIgnoreMissing: &False,
				}, u.ID); err != nil {
					srv.Logger.Errorf("certificate fe api: update domain %s: certificate ignore missing false: %s", domain.FQDN, err)
					time.Sleep(60 * time.Second)
					continue
				}
				return
			}
		}()
	}

	certificate, err := srv.CertificateService.ObtainCertificate(domain.FQDN)
	if err != nil {
		srv.Logger.Warningf("certificate fe api: obtain certificate: %s: %s", domain.FQDN, err)
		jsonresponse.BadRequest(w, web.NewError("Unable to obtain TLS certificate."))
		return
	}
	srv.Logger.Infof("certificate api: obtain certificate: success for %s: expiration time: %s", certificate.FQDN, certificate.ExpirationTime)

	auditf(r, nil, "obtain certificate", "%s: %s", certificate.FQDN, certificate.ExpirationTime)

	jsonresponse.OK(w, nil)
}

type domainFEAPIRequest struct {
	FQDN              string           `json:"fqdn"`
	CertificateIgnore marshal.Checkbox `json:"certificateIgnore"`
	Disabled          marshal.Checkbox `json:"disabled"`
}

type domainToken struct {
	FQDN  string `json:"fqdn"`
	Token string `json:"token"`
}

type validationFormErrorResponse struct {
	web.FormErrors
	Tokens []domainToken `json:"tokens"`
}

func domainFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	request := domainFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("domain fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, web.NewError("Invalid data."))
		return
	}

	fqdn := strings.TrimSpace(request.FQDN)
	if fqdn == "" {
		srv.Logger.Warning("domain fe api: request: fqdn empty")
		jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Fully qualified domain name is required."))
		return
	}

	if !fqdnRegex.MatchString(fqdn) && fqdn != srv.Domain {
		jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Fully qualified domain name is invalid."))
		return
	}

	var domain *packages.Domain
	if id != "" {
		domain, err = srv.PackagesService.Domain(id)
		if err != nil {
			switch err {
			case packages.ErrDomainNotFound:
				jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
				return
			case nil:
			default:
				srv.Logger.Errorf("domain fe api: domain %s: %s", id, err)
				jsonServerError(w, err)
				return
			}
		}
	}

	for _, d := range srv.ForbiddenDomains {
		if d == fqdn || strings.HasSuffix(fqdn, "."+d) {
			jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Domain is not available"))
			return
		}
	}

	skipDomainVerification := srv.SkipDomainVerification
	if !skipDomainVerification {
		for _, d := range srv.TrustedDomains {
			if fqdn == d || strings.HasSuffix(fqdn, "."+d) {
				skipDomainVerification = true
				break
			}
		}
	}

	// New or changed domain fqdn verification.
	if (domain == nil || domain.FQDN != fqdn) && srv.Domain != "" && !skipDomainVerification {
		switch {
		case fqdn == srv.Domain, strings.HasSuffix(fqdn, "."+srv.Domain):
			if strings.Count(fqdn, ".") > strings.Count(srv.Domain, ".")+1 {
				jsonresponse.BadRequest(w, web.NewFieldError("fqdn", fmt.Sprintf("Only one subdomain is allowed for domain %s", srv.Domain)))
				return
			}
		default:
			publicSuffix, icann := publicsuffix.PublicSuffix(fqdn)
			if !icann {
				jsonresponse.BadRequest(w, web.NewFieldError("fqdn", fmt.Sprintf("Top level domain %s is not an ICANN domain.", publicSuffix)))
				return
			}
			if fqdn == publicSuffix {
				jsonresponse.BadRequest(w, web.NewFieldError("fqdn", fmt.Sprintf("The domain %s is an ICANN domain.", publicSuffix)))
				return
			}

			tokens := []domainToken{}
			domainParts := strings.Split(fqdn, ".")
			startIndex := len(domainParts) - strings.Count(publicSuffix, ".") - 2
			if startIndex < 0 {
				jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Fully qualified domain name is invalid."))
				return
			}

			domain, err = srv.PackagesService.Domain(fqdn)
			if err != nil {
				if err != packages.ErrDomainNotFound {
					srv.Logger.Errorf("domain fe api: domain %s: %s", fqdn, err)
					jsonServerError(w, err)
					return
				}
			} else {
				jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Domain already exists."))
				return
			}

			d := publicSuffix
			var token, verificationDomain string
			var verified bool
			var x [20]byte
			for i := startIndex; i >= 0; i-- {
				d = fmt.Sprintf("%s.%s", domainParts[i], d)

				x = sha1.Sum(append(srv.salt, []byte(u.ID+d)...))
				token = base64.URLEncoding.EncodeToString(x[:])

				verificationDomain = srv.VerificationSubdomain + "." + d

				verified, err = verifyDomain(verificationDomain, token)
				if err != nil {
					srv.Logger.Errorf("domain fe api: verify domain: %s: %s", verificationDomain, err)
				}
				if verified {
					break
				}

				tokens = append(tokens, domainToken{
					FQDN:  verificationDomain,
					Token: token,
				})
			}

			if !verified {
				jsonresponse.BadRequest(w, validationFormErrorResponse{
					FormErrors: web.NewFieldError("fqdn", "Domain is not verified."),
					Tokens:     tokens,
				})
				return
			}
		}
	}

	disabled := request.Disabled.Bool()

	var editedDomain *packages.Domain
	if id == "" {
		t := true
		editedDomain, err = srv.PackagesService.AddDomain(&packages.DomainOptions{
			FQDN:        &request.FQDN,
			OwnerUserID: &u.ID,
			Disabled:    &disabled,

			CertificateIgnoreMissing: &t,
		}, u.ID)
	} else {
		certificateIgnore := request.CertificateIgnore.Bool()
		editedDomain, err = srv.PackagesService.UpdateDomain(id, &packages.DomainOptions{
			FQDN:              &request.FQDN,
			CertificateIgnore: &certificateIgnore,
			Disabled:          &disabled,
		}, u.ID)
	}
	if err != nil {
		switch err {
		case packages.ErrDomainFQDNRequired:
			jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Domain fully qualified domain name is required."))
			return
		case packages.ErrDomainOwnerUserIDRequired:
			jsonresponse.BadRequest(w, web.NewError("Domain user is required."))
			return
		case packages.ErrDomainNotFound:
			jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Unknown domain."))
			return
		case packages.ErrDomainAlreadyExists:
			jsonresponse.BadRequest(w, web.NewFieldError("fqdn", "Domain is already registered."))
			return
		case nil:
		default:
			srv.Logger.Errorf("domain fe api: add/update domain %s: %s", id, err)
			jsonServerError(w, err)
			return
		}
	}

	// Obtain certificate only if:
	// - it is the new domain (id == "")
	// - TLS server is active (s.tlsEnabled == true)
	// - the domain as actually created (editedDomain != nil)
	if id == "" && srv.tlsEnabled && editedDomain != nil {
		go func() {
			defer srv.RecoveryService.Recover()
			defer func() {
				f := false
				for {
					if _, err = srv.PackagesService.UpdateDomain(editedDomain.ID, &packages.DomainOptions{
						CertificateIgnoreMissing: &f,
					}, u.ID); err != nil {
						srv.Logger.Errorf("domain fe api: update domain %s: certificate ignore missing false: %s", editedDomain.FQDN, err)
						time.Sleep(60 * time.Second)
						continue
					}
					return
				}
			}()
			certificate, err := srv.CertificateService.ObtainCertificate(editedDomain.FQDN)
			if err != nil {
				srv.Logger.Errorf("domain fe api: obtain certificate: %s: %s", editedDomain.FQDN, err)
				return
			}
			srv.Logger.Infof("domain fe api: obtain certificate: success for %s: expiration time: %s", certificate.FQDN, certificate.ExpirationTime)
		}()
	}

	action := "domain update"
	if id == "" {
		action = "domain add"
	}
	auditf(r, request, action, "%s: %s", editedDomain.ID, editedDomain.FQDN)

	jsonresponse.OK(w, editedDomain)
}

func domainDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	deletedDomain, err := srv.PackagesService.DeleteDomain(id, u.ID)
	if err != nil {
		switch err {
		case packages.ErrDomainNotFound:
			jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
			return
		case nil:
		default:
			srv.Logger.Errorf("domain delete fe api: domain %s: %s", id, err)
			jsonServerError(w, err)
			return
		}
	}

	srv.Logger.Debugf("domain delete fe api: %s deleted by %s", deletedDomain.ID, u.ID)

	auditf(r, nil, "domain delete", "%s: %s", deletedDomain.ID, deletedDomain.FQDN)

	jsonresponse.OK(w, deletedDomain)
}

type userIDRequest struct {
	ID string `json:"id"`
}

func domainUserGrantFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userIDRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("domain user grant fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, web.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		jsonresponse.BadRequest(w, web.NewFieldError("id", "User is required."))
		return
	}

	domainID := mux.Vars(r)["id"]

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, web.NewFieldError("id", "You are already granted."))
		return
	}

	grantUser, err := srv.UserService.User(request.ID)
	if err != nil {
		if err == user.ErrUserNotFound {
			srv.Logger.Warningf("domain user grant fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, web.NewFieldError("id", "Unknown user."))
			return
		}
		srv.Logger.Errorf("domain user grant fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	err = srv.PackagesService.AddUserToDomain(domainID, grantUser.ID, u.ID)
	switch err {
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case packages.ErrUserExists:
		jsonresponse.BadRequest(w, web.NewFieldError("id", "This user is already granted."))
		return
	case packages.ErrForbidden:
		jsonresponse.BadRequest(w, web.NewError("You do not have permission to revoke user."))
		return
	case nil:
	default:
		srv.Logger.Errorf("domain user grant fe api: add user to domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	audit(r, grantUser.ID, "domain user grant", domainID)

	jsonresponse.OK(w, nil)
}

func domainUserRevokeFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userIDRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("domain user grant fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, web.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		jsonresponse.BadRequest(w, web.NewFieldError("id", "User is required."))
		return
	}

	domainID := mux.Vars(r)["id"]

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, web.NewFieldError("id", "You can not revoke yourself."))
		return
	}

	revokeUser, err := srv.UserService.User(request.ID)
	if err != nil {
		if err == user.ErrUserNotFound {
			srv.Logger.Warningf("domain user revoke fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, web.NewFieldError("id", "Unknown user."))
			return
		}
		srv.Logger.Errorf("domain user revoke fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	err = srv.PackagesService.RemoveUserFromDomain(domainID, revokeUser.ID, u.ID)
	switch err {
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case packages.ErrUserDoesNotExist:
		jsonresponse.BadRequest(w, web.NewFieldError("id", "This user is not granted."))
		return
	case packages.ErrForbidden:
		jsonresponse.BadRequest(w, web.NewError("You do not have permission to revoke user."))
		return
	case nil:
	default:
		srv.Logger.Errorf("domain user revoke fe api: revoke user form domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	audit(r, revokeUser.ID, "domain user revoke", domainID)

	jsonresponse.OK(w, nil)
}

type domainOwnerChangeFEAPIRequest struct {
	ID string `json:"id"`
}

func domainOwnerChangeFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	domainID := mux.Vars(r)["id"]

	request := domainOwnerChangeFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("domain owner change fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, web.NewError("Invalid data."))
		return
	}

	if request.ID == "" {
		srv.Logger.Warning("domain owner change fe api: request: id empty")
		jsonresponse.BadRequest(w, web.NewFieldError("id", "User ID is required."))
		return
	}

	domain, err := srv.PackagesService.Domain(domainID)
	if err != nil {
		switch err {
		case packages.ErrDomainNotFound:
			jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
			return
		case nil:
		default:
			srv.Logger.Errorf("domain fe api: domain %s: %s", domainID, err)
			jsonServerError(w, err)
			return
		}
	}

	if domain.OwnerUserID != u.ID {
		jsonresponse.BadRequest(w, web.NewError("You do not have permission to change the owner of this domain."))
		return
	}

	if request.ID == u.Username || request.ID == u.Email || request.ID == u.ID {
		jsonresponse.BadRequest(w, web.NewFieldError("id", "You are already the owner."))
		return
	}

	owner, err := srv.UserService.User(request.ID)
	if err != nil {
		if err == user.ErrUserNotFound {
			srv.Logger.Warningf("domain owner change fe api: user %s: %s", request.ID, err)
			jsonresponse.BadRequest(w, web.NewFieldError("id", "Unknown user."))
			return
		}
		srv.Logger.Errorf("domain user revoke fe api: user %s: %s", request.ID, err)
		jsonServerError(w, err)
		return
	}
	domain, err = srv.PackagesService.UpdateDomain(domainID, &packages.DomainOptions{
		OwnerUserID: &owner.ID,
	}, u.ID)
	switch err {
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case nil:
	default:
		srv.Logger.Errorf("domain owner change fe api: update domain %s: %s", domainID, err)
		jsonServerError(w, err)
		return
	}

	auditf(r, request, "domain owner change", "%s: %s to %s: %s", domain.ID, domain.FQDN, owner.ID, owner.Email)

	jsonresponse.OK(w, domain)
}

type packageFEAPIRequest struct {
	DomainID    string           `json:"domainId"`
	Path        string           `json:"path"`
	VCS         packages.VCS     `json:"vcs"`
	RepoRoot    string           `json:"repoRoot"`
	RefType     packages.RefType `json:"refType"`
	RefName     string           `json:"refName"`
	GoSource    string           `json:"goSource"`
	RedirectURL string           `json:"redirectUrl"`
	Disabled    marshal.Checkbox `json:"disabled"`
}

func packageFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	request := packageFEAPIRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		srv.Logger.Warningf("package fe api: request decode: %s", err)
		jsonresponse.BadRequest(w, web.NewError("Invalid data."))
		return
	}

	errors := web.FormErrors{}

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
			srv.Logger.Warningf("package fe api: %s %s: invalid repository url: %s: %s", request.DomainID, request.Path, request.RepoRoot, err)
			errors.AddFieldError("repoRoot", "Invalid Repository URL.")
		case request.VCS != "":
			if repoRoot.Scheme == "" {
				errors.AddFieldError("repoRoot", fmt.Sprintf("Repository URL requires a URL scheme (%s).", strings.Join(packages.VCSSchemes[request.VCS], ", ")))
				break
			}
			ok := false
			for _, s := range packages.VCSSchemes[request.VCS] {
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
				errors.AddFieldError("repoRoot", fmt.Sprintf("Invalid scheme \"%s\". For %s repository it should be one of (%s).", repoRoot.Scheme, vcs, strings.Join(packages.VCSSchemes[request.VCS], ", ")))
			}
			if !hostAndPortRegex.MatchString(repoRoot.Host) {
				errors.AddFieldError("repoRoot", fmt.Sprintf("Invalid domain and port \"%s\".", repoRoot.Host))
			}
		}
	}

	switch request.RefType {
	case "", packages.RefTypeTag, packages.RefTypeBranch:
	default:
		errors.AddFieldError("refType", "Reference type must be branch or tag.")
	}

	if request.RefType != "" && request.RefName == "" {
		errors.AddFieldError("refName", "Reference name is required if reference type is selected.")
	}

	if request.RefName != "" && request.VCS != "" && (request.VCS != packages.VCSGit || (request.VCS == packages.VCSGit && repoRoot != nil && !(repoRoot.Scheme == "http" || repoRoot.Scheme == "https"))) {
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
		p, err = srv.PackagesService.AddPackage(&packages.PackageOptions{
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
		p, err = srv.PackagesService.UpdatePackage(id, &packages.PackageOptions{
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
	case packages.ErrForbidden:
		jsonresponse.BadRequest(w, web.NewError("You do not have permission to add packages to this domain."))
		return
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case packages.ErrPackageNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown package."))
		return
	case packages.ErrPackageDomainRequired:
		jsonresponse.BadRequest(w, web.NewError("Domain is required."))
		return
	case packages.ErrPackagePathRequired:
		jsonresponse.BadRequest(w, web.NewFieldError("path", "Path is required."))
		return
	case packages.ErrPackageVCSRequired:
		jsonresponse.BadRequest(w, web.NewFieldError("vcs", "VCS is required."))
		return
	case packages.ErrPackageRepoRootRequired:
		jsonresponse.BadRequest(w, web.NewFieldError("repoRoot", "Repository is required."))
		return
	case packages.ErrPackageRepoRootInvalid:
		jsonresponse.BadRequest(w, web.NewFieldError("repoRoot", "Repository is invalid."))
		return
	case packages.ErrPackageRepoRootSchemeRequired:
		jsonresponse.BadRequest(w, web.NewFieldError("repoRoot", "Repository URL scheme is required."))
		return
	case packages.ErrPackageRepoRootSchemeInvalid:
		jsonresponse.BadRequest(w, web.NewFieldError("repoRoot", "Repository URL scheme is invalid."))
		return
	case packages.ErrPackageRepoRootHostInvalid:
		jsonresponse.BadRequest(w, web.NewFieldError("repoRoot", "Repository URL host is invalid."))
		return
	case packages.ErrPackageRefChangeRejected:
		jsonresponse.BadRequest(w, web.NewFieldError("refName", "Reference change is allowed only for Git HTTP and HTTPS repositeries."))
		return
	case packages.ErrPackageAlreadyExists:
		jsonresponse.BadRequest(w, web.NewFieldError("path", "Package already exists."))
		return
	case nil:
	default:
		srv.Logger.Errorf("package fe api: add/update package %s: %s", id, err)
		jsonServerError(w, err)
		return
	}

	action := "package update"
	if id == "" {
		action = "package add"
	}
	auditf(r, request, action, "%s %s (domain: %s)", p.ID, p.ImportPrefix(), p.Domain.ID)

	jsonresponse.OK(w, p)
}

func packageDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	// Delete package checks permissions.
	p, err := srv.PackagesService.DeletePackage(id, u.ID)
	switch err {
	case packages.ErrForbidden:
		jsonresponse.BadRequest(w, web.NewError("You do not have permission to add packages to this domain."))
		return
	case packages.ErrDomainNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown domain."))
		return
	case packages.ErrPackageNotFound:
		jsonresponse.BadRequest(w, web.NewError("Unknown package."))
		return
	case nil:
	default:
		srv.Logger.Errorf("package delete fe api: delete package %s: %s", id, err)
		jsonServerError(w, err)
		return
	}

	srv.Logger.Debugf("package delete fe api: %s deleted by %s", p.ID, u.ID)

	auditf(r, nil, "package delete", "%s: %s", p.ID, p.ImportPrefix)

	jsonresponse.OK(w, p)
}
