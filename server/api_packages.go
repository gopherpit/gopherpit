// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"resenje.org/jsonresponse"

	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/services/packages"
)

func (s Server) packageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	p, err := s.PackagesService.Package(id)
	if err != nil {
		if err == packages.PackageNotFound {
			s.logger.Warningf("package api: package %s: %s", id, err)
			jsonresponse.NotFound(w, api.PackageNotFound)
			return
		}
		s.logger.Errorf("package api: package %s: %s", id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	token := ""
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("package api: domains by user %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("package api: domains by user %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("package api: domains by user %s: %s", u.ID, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
		for _, d := range response.Domains {
			if p.Domain.ID == d.ID {
				authorized = true
				break
			}
		}
		token = response.Next
		if token == "" || authorized {
			break
		}
	}

	if !authorized {
		s.logger.Errorf("package api: package %s: does not belong to user %s", id, u.ID)
		jsonresponse.Forbidden(w, nil)
		return
	}

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func (s Server) updatePackageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	warningf := func(format string, a ...interface{}) {
		s.logger.Warningf("update package api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}
	errorf := func(format string, a ...interface{}) {
		s.logger.Errorf("update package api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}

	request := api.PackageOptions{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		warningf("request: request decode: %s", err)
		jsonresponse.BadRequest(w, nil)
		return
	}

	if request.Domain == nil || *request.Domain == "" {
		warningf("request: domain absent")
		jsonresponse.BadRequest(w, api.PackageDomainRequired)
		return
	}

	if request.Path == nil || *request.Path == "" {
		warningf("request: path absent")
		jsonresponse.BadRequest(w, api.PackagePathRequired)
		return
	}

	if request.VCS == nil || *request.VCS == "" {
		warningf("request: vcs absent")
		jsonresponse.BadRequest(w, api.PackageVCSRequired)
		return
	}

	var repoRoot *url.URL
	if request.RepoRoot == nil || *request.RepoRoot == "" {
		warningf("request: repo root absent")
		jsonresponse.BadRequest(w, api.PackageRepoRootRequired)
		return
	}

	repoRoot, err = url.Parse(*request.RepoRoot)
	switch {
	case err != nil:
		warningf("request: parse repo root: %s", err)
		jsonresponse.BadRequest(w, api.PackageRepoRootInvalid)
		return
	case request.VCS != nil && *request.VCS != "":
		if repoRoot.Scheme == "" {
			warningf("repo root: missing url scheme")
			jsonresponse.BadRequest(w, api.PackageRepoRootSchemeRequired)
			return
		}
		ok := false
		for _, s := range vcsSchemas[packages.VCS(*request.VCS)] {
			if repoRoot.Scheme == s {
				ok = true
				break
			}
		}
		if !ok {
			warningf("repo root: invalid url scheme %q", repoRoot.Scheme)
			jsonresponse.BadRequest(w, api.PackageRepoRootSchemeInvalid)
			return
		}
		if !hostAndPortRegex.MatchString(repoRoot.Host) {
			warningf("repo root: invalid url host %q", repoRoot.Host)
			jsonresponse.BadRequest(w, api.PackageRepoRootHostInvalid)
			return
		}
	}

	refType := ""
	if request.RefType != nil {
		refType = *request.RefType
	}

	refName := ""
	if request.RefName != nil {
		refName = *request.RefName
	}

	switch refType {
	case "", packages.RefTypeTag, packages.RefTypeBranch:
	default:
		warningf("invalid reference type %q", refType)
		jsonresponse.BadRequest(w, api.PackageRefTypeInvalid)
		return
	}

	if refType != "" && refName == "" {
		warningf("missing reference name")
		jsonresponse.BadRequest(w, api.PackageRefNameRequired)
		return
	}

	if refName != "" && (packages.VCS(*request.VCS) != packages.VCSGit || (packages.VCS(*request.VCS) == packages.VCSGit && repoRoot != nil && !(repoRoot.Scheme == "http" || repoRoot.Scheme == "https"))) {
		warningf("reference change rejected")
		jsonresponse.BadRequest(w, api.PackageRefChangeRejected)
		return
	}

	if request.RedirectURL != nil && !urlRegex.MatchString(*request.RedirectURL) {
		warningf("invalid redirect url: %s", request.RedirectURL)
		jsonresponse.BadRequest(w, api.PackageRedirectURLInvalid)
		return
	}

	if !strings.HasPrefix(*request.Path, "/") {
		*request.Path = "/" + *request.Path
	}

	vcs := packages.VCS(*request.VCS)

	o := &packages.PackageOptions{
		Domain:      request.Domain,
		Path:        request.Path,
		VCS:         &vcs,
		RepoRoot:    request.RepoRoot,
		RefType:     request.RefType,
		RefName:     request.RefName,
		GoSource:    request.GoSource,
		RedirectURL: request.RedirectURL,
		Disabled:    request.Disabled,
	}
	var p *packages.Package
	if id == "" {
		p, err = s.PackagesService.AddPackage(o, u.ID)
	} else {
		p, err = s.PackagesService.UpdatePackage(id, o, u.ID)
	}
	switch err {
	case packages.Forbidden:
		warningf("add/update package: %s", err)
		jsonresponse.Forbidden(w, nil)
		return
	case packages.DomainNotFound:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.DomainNotFound)
		return
	case packages.PackageNotFound:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackageNotFound)
		return
	case packages.PackageDomainRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackageDomainRequired)
		return
	case packages.PackagePathRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackagePathRequired)
		return
	case packages.PackageVCSRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackageVCSRequired)
		return
	case packages.PackageRepoRootRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackageRepoRootRequired)
		return
	case packages.PackageAlreadyExists:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.PackageAlreadyExists)
		return
	case nil:
	default:
		errorf("add/update package: %s", err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	action := "package update"
	if id == "" {
		action = "package add"
	}
	s.auditf(r, request, action, "%s %s (domain: %s)", p.ID, p.ImportPrefix(), p.Domain.ID)

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func (s Server) deletePackageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	// Delete package checks permissions.
	p, err := s.PackagesService.DeletePackage(id, u.ID)
	switch err {
	case packages.Forbidden:
		s.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.Forbidden(w, nil)
		return
	case packages.DomainNotFound:
		s.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.BadRequest(w, api.DomainNotFound)
		return
	case packages.PackageNotFound:
		s.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.BadRequest(w, api.PackageNotFound)
		return
	case nil:
	default:
		s.logger.Errorf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	s.logger.Debugf("package delete api: %s deleted by %s", p.ID, u.ID)

	s.auditf(r, nil, "package delete", "%s: %s", p.ID, p.ImportPrefix)

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func (s Server) domainPackagesAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	start := r.URL.Query().Get("start")

	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		var err error
		limit, err = strconv.Atoi(l)
		if err != nil || limit < 1 || limit > api.MaxLimit {
			limit = 0
			return
		}
	}

	pkgs, err := s.PackagesService.PackagesByDomain(id, start, limit)
	if err != nil {
		if err == packages.DomainNotFound {
			s.logger.Warningf("domain packages api: packages by domain %s: %s", id, err)
			jsonresponse.BadRequest(w, api.DomainNotFound)
			return
		}
		s.logger.Errorf("domain packages api: packages by domain %s: %s", id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	token := ""
	authorized := false
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				s.logger.Warningf("domain packages api: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				s.logger.Warningf("domain packages api: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("domain packages api: user domains %s: %s", u.ID, err)
			jsonresponse.InternalServerError(w, nil)
			return
		}
		for _, domain := range response.Domains {
			if domain.ID == pkgs.Domain.ID {
				authorized = true
			}
		}
		token = response.Next
		if token == "" || authorized {
			break
		}
	}

	if !authorized {
		s.logger.Errorf("domain packages api: domain %s: not allowed for user %s", id, u.ID)
		jsonresponse.Forbidden(w, nil)
		return
	}

	response := api.PackagesPage{
		Packages: api.Packages{},
		Previous: pkgs.Previous,
		Next:     pkgs.Next,
		Count:    pkgs.Count,
	}

	for _, p := range pkgs.Packages {
		response.Packages = append(response.Packages, packagesPackageToAPIPackage(p, pkgs.Domain))
	}

	jsonresponse.OK(w, response)
}
