// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

func (s Server) domainPackagesHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]
	start := r.URL.Query().Get("start")
	if start != "" {
		start, err = func(text string) (string, error) {
			m := len(text) % 8
			if m != 0 {
				text = text + strings.Repeat("=", 8-m)
			}
			b, err := base32.StdEncoding.DecodeString(text)
			return string(b), err
		}(start)
		if err != nil {
			s.logger.Warningf("domain packages: base32 decode start %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
	}
	pkgs, err := s.PackagesService.PackagesByDomain(id, start, 0)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain packages: packages by domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain packages: packages by domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain packages: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain packages: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain packages: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, domain := range response.Domains {
			if !authorized && domain.ID == pkgs.Domain.ID {
				authorized = true
			}
			domains = append(domains, domain)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	respond(w, s.templateDomainPackages(), map[string]interface{}{
		"User":     u,
		"Domain":   pkgs.Domain,
		"Domains":  domains,
		"Packages": pkgs.Packages,
		"Previous": pkgs.Previous,
		"Next":     pkgs.Next,
	})
}

func (s Server) domainAddHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	domain := packages.Domain{}
	if id != "" {
		d, err := s.PackagesService.Domain(id)
		if err != nil {
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain add: domain %s: %s", id, err)
				s.htmlNotFoundHandler(w, r)
				return
			}
			s.logger.Errorf("domain add: domain %s: %s", id, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		domain = *d
		if domain.OwnerUserID != u.ID {
			s.htmlNotFoundHandler(w, r)
			return
		}
	}

	respond(w, s.templateDomainAdd(), map[string]interface{}{
		"User":       u,
		"Domain":     domain,
		"DomainName": s.Domain,
	})
}

func (s Server) domainChangelogHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain changelog: domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain changelog: domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain changelog: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain changelog: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain changelog: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, d := range response.Domains {
			if !authorized && domain.ID == d.ID {
				authorized = true
			}
			domains = append(domains, d)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	start := r.URL.Query().Get("start")

	changelog, err := s.PackagesService.ChangelogForDomain(id, start, 20)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain changelog: domain changelog %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain changelog: domain changelog %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}
	if start != "" && changelog.Next == "" {
		q := r.URL.Query()
		q.Del("start")
		r.URL.RawQuery = q.Encode()
		http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
		return
	}

	users := map[string]*user.User{
		u.ID: u,
	}
	records := make([]changelogRecord, 0, len(changelog.Records))
	for _, record := range changelog.Records {
		if err = s.updateChangelogRecords(*u, record, &records, &users); err != nil {
			s.logger.Errorf("domain changelog: update users map: %s", err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
	}

	respond(w, s.templateDomainChangelog(), map[string]interface{}{
		"User":             u,
		"Domain":           domain,
		"Domains":          domains,
		"ChangelogRecords": records,
		"Next":             changelog.Next,
		"Previous":         changelog.Previous,
	})
}

func (s Server) domainTeamHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	var domain *packages.Domain
	domain, err = s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain team: domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain team: domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain team: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain team: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain team: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		domains = append(domains, response.Domains...)
		token = response.Next
		if token == "" {
			break
		}
	}

	if domain.OwnerUserID != u.ID {
		respond(w, s.templateDomainTeam(), map[string]interface{}{
			"Forbidden": true,
			"User":      u,
			"Domain":    domain,
			"Domains":   domains,
		})
		return
	}

	domainUsers, err := s.PackagesService.DomainUsers(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain team: domain users %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain team: domain users %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	users := []user.User{}
	for _, id := range domainUsers.UserIDs {
		domainUser, err := s.UserService.User(id)
		if err != nil {
			if err == user.UserNotFound {
				s.logger.Warningf("domain team: user %s: %s", id, err)
				continue
			}
			s.logger.Errorf("domain team: user %s: %s", id, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		if domainUser == nil || domainUser.Disabled {
			continue
		}
		users = append(users, *domainUser)
	}

	respond(w, s.templateDomainTeam(), map[string]interface{}{
		"Forbidden": false,
		"User":      u,
		"Users":     users,
		"Domain":    domain,
		"Domains":   domains,
	})
}

func (s Server) domainSettingsHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	var domain *packages.Domain
	if id != "" {
		domain, err = s.PackagesService.Domain(id)
		if err != nil {
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain settings: domain %s: %s", id, err)
				s.htmlNotFoundHandler(w, r)
				return
			}
			s.logger.Errorf("domain settings: domain %s: %s", id, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
	}
	token := ""
	domains := packages.Domains{}
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain settings: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain settings: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain settings: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		domains = append(domains, response.Domains...)
		token = response.Next
		if token == "" {
			break
		}
	}

	isCertificateBeingObtained, err := s.CertificateService.IsCertificateBeingObtained(domain.FQDN)
	if err != nil {
		s.logger.Errorf("domain settings: is certificate being optained %s: %s", domain.FQDN, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	var certificateExpirationTime *time.Time
	if !isCertificateBeingObtained {
		cert, err := s.CertificateService.Certificate(domain.FQDN)
		switch err {
		case certificate.CertificateNotFound:
		case nil:
			certificateExpirationTime = cert.ExpirationTime
		default:
			s.logger.Errorf("domain settings: certificate %s: %s", domain.FQDN, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
	}

	respond(w, s.templateDomainSettings(), map[string]interface{}{
		"Forbidden":                  domain.OwnerUserID != u.ID,
		"User":                       u,
		"Domain":                     domain,
		"Domains":                    domains,
		"CertificateExpirationTime":  certificateExpirationTime,
		"IsCertificateBeingObtained": isCertificateBeingObtained,
	})
}

func (s Server) domainDomainUserGrantHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain user grant: domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain user grant: domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain user grant: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain user grant: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain user grant: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, d := range response.Domains {
			if !authorized && domain.ID == d.ID {
				authorized = true
			}
			domains = append(domains, d)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	respond(w, s.templateDomainUserGrant(), map[string]interface{}{
		"User":    u,
		"Domain":  domain,
		"Domains": domains,
	})
}

func (s Server) domainDomainUserRevokeHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain user revoke: domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain user revoke: domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain user revoke: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain user revoke: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain user revoke: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, d := range response.Domains {
			if !authorized && domain.ID == d.ID {
				authorized = true
			}
			domains = append(domains, d)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	userID := vars["user-id"]
	domainUser, err := s.UserService.User(userID)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("domain user revoke: user %s: %s", userID, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain user revoke: user %s: %s", userID, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	domainUsers, err := s.PackagesService.DomainUsers(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain user revoke: domain users %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain user revoke: domain users %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	found := false
	for _, id := range domainUsers.UserIDs {
		if id == domainUser.ID {
			found = true
			break
		}
	}
	if !found {
		s.htmlNotFoundHandler(w, r)
		return
	}

	respond(w, s.templateDomainUserRevoke(), map[string]interface{}{
		"User":       u,
		"Domain":     domain,
		"Domains":    domains,
		"DomainUser": domainUser,
	})
}

func (s Server) domainDomainOwnerChangeHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	domain, err := s.PackagesService.Domain(id)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain owner change: domain %s: %s", id, err)
			s.htmlNotFoundHandler(w, r)
			return
		}
		s.logger.Errorf("domain owner change: domain %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain owner change: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain owner change: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain owner change: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, d := range response.Domains {
			if !authorized && domain.ID == d.ID {
				authorized = true
			}
			domains = append(domains, d)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	respond(w, s.templateDomainOwnerChange(), map[string]interface{}{
		"User":    u,
		"Domain":  domain,
		"Domains": domains,
	})
}

type vcsInfo struct {
	VCS  packages.VCS
	Name string
}

var vcsInfos = []vcsInfo{
	vcsInfo{packages.VCSGit, "Git"},
	vcsInfo{packages.VCSMercurial, "Mercurial"},
	vcsInfo{packages.VCSBazaar, "Bazaar"},
	vcsInfo{packages.VCSSubversion, "Subversion"},
}

func (s Server) domainPackageEditHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	var pkg packages.Package
	var domain *packages.Domain
	vars := mux.Vars(r)
	domainID := vars["domain-id"]
	packageID := vars["package-id"]
	if packageID != "" {
		p, err := s.PackagesService.Package(packageID)
		if err != nil {
			if err == packages.PackageNotFound {
				s.logger.Warningf("domain package edit: package %s: %s", packageID, err)
				s.htmlNotFoundHandler(w, r)
				return
			}
			s.logger.Errorf("domain package edit: package %s: %s", packageID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		pkg = *p
		domain = pkg.Domain
	} else {
		domain, err = s.PackagesService.Domain(domainID)
		if err != nil {
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain package edit: domain %s: %s", domainID, err)
				s.htmlNotFoundHandler(w, r)
				return
			}
			s.logger.Errorf("domain package edit: domain %s: %s", domainID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
	}

	token := ""
	domains := packages.Domains{}
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain package edit: domains by user %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain package edit: domains by user %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain package edit: domains by user %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		for _, d := range response.Domains {
			if !authorized && domain.ID == d.ID {
				authorized = true
			}
			domains = append(domains, d)
		}
		token = response.Next
		if token == "" {
			break
		}
	}

	if !authorized {
		s.htmlNotFoundHandler(w, r)
		return
	}

	respond(w, s.templateDomainPackageEdit(), map[string]interface{}{
		"User":     u,
		"Domain":   domain,
		"Domains":  domains,
		"Package":  pkg,
		"VCSInfos": vcsInfos,
	})
}

func (s Server) userPageHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	pu, err := s.UserService.UserByID(id)
	if err != nil {
		if err == user.UserNotFound {
			s.logger.Warningf("user page: user by id %s: %s", id, err)
			return
		}
		s.logger.Errorf("user page: user by id %s: %s", id, err)
		s.htmlInternalServerErrorHandler(w, r)
		return
	}

	respond(w, s.templateUserPage(), map[string]interface{}{
		"User":     u,
		"PageUser": pu,
	})
}
