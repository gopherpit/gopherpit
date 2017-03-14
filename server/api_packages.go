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

func packageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	vars := mux.Vars(r)
	id := vars["id"]

	p, err := srv.PackagesService.Package(id)
	if err != nil {
		if err == packages.PackageNotFound {
			srv.logger.Warningf("package api: package %s: %s", id, err)
			jsonresponse.NotFound(w, api.ErrPackageNotFound)
			return
		}
		srv.logger.Errorf("package api: package %s: %s", id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	token := ""
	authorized := false
	for {
		response, err := srv.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				srv.logger.Warningf("package api: domains by user %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				srv.logger.Warningf("package api: domains by user %s: %s", u.ID, err)
				break
			}
			srv.logger.Errorf("package api: domains by user %s: %s", u.ID, err)
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
		srv.logger.Errorf("package api: package %s: does not belong to user %s", id, u.ID)
		jsonresponse.Forbidden(w, nil)
		return
	}

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func updatePackageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	warningf := func(format string, a ...interface{}) {
		srv.logger.Warningf("update package api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}
	errorf := func(format string, a ...interface{}) {
		srv.logger.Errorf("update package api: %q: user %s: %s", id, u.ID, fmt.Sprintf(format, a...))
	}

	request := api.PackageOptions{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		warningf("request: request decode: %s", err)
		jsonresponse.BadRequest(w, nil)
		return
	}

	if request.Domain == nil || *request.Domain == "" {
		warningf("request: domain absent")
		jsonresponse.BadRequest(w, api.ErrPackageDomainRequired)
		return
	}

	if request.Path == nil || *request.Path == "" {
		warningf("request: path absent")
		jsonresponse.BadRequest(w, api.ErrPackagePathRequired)
		return
	}

	if request.VCS == nil || *request.VCS == "" {
		warningf("request: vcs absent")
		jsonresponse.BadRequest(w, api.ErrPackageVCSRequired)
		return
	}

	var repoRoot *url.URL
	if request.RepoRoot == nil || *request.RepoRoot == "" {
		warningf("request: repo root absent")
		jsonresponse.BadRequest(w, api.ErrPackageRepoRootRequired)
		return
	}

	repoRoot, err = url.Parse(*request.RepoRoot)
	switch {
	case err != nil:
		warningf("request: parse repo root: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageRepoRootInvalid)
		return
	case request.VCS != nil && *request.VCS != "":
		if repoRoot.Scheme == "" {
			warningf("repo root: missing url scheme")
			jsonresponse.BadRequest(w, api.ErrPackageRepoRootSchemeRequired)
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
			jsonresponse.BadRequest(w, api.ErrPackageRepoRootSchemeInvalid)
			return
		}
		if !hostAndPortRegex.MatchString(repoRoot.Host) {
			warningf("repo root: invalid url host %q", repoRoot.Host)
			jsonresponse.BadRequest(w, api.ErrPackageRepoRootHostInvalid)
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
		jsonresponse.BadRequest(w, api.ErrPackageRefTypeInvalid)
		return
	}

	if refType != "" && refName == "" {
		warningf("missing reference name")
		jsonresponse.BadRequest(w, api.ErrPackageRefNameRequired)
		return
	}

	if refName != "" && (packages.VCS(*request.VCS) != packages.VCSGit || (packages.VCS(*request.VCS) == packages.VCSGit && repoRoot != nil && !(repoRoot.Scheme == "http" || repoRoot.Scheme == "https"))) {
		warningf("reference change rejected")
		jsonresponse.BadRequest(w, api.ErrPackageRefChangeRejected)
		return
	}

	if request.RedirectURL != nil && !urlRegex.MatchString(*request.RedirectURL) {
		warningf("invalid redirect url: %s", request.RedirectURL)
		jsonresponse.BadRequest(w, api.ErrPackageRedirectURLInvalid)
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
		p, err = srv.PackagesService.AddPackage(o, u.ID)
	} else {
		p, err = srv.PackagesService.UpdatePackage(id, o, u.ID)
	}
	switch err {
	case packages.Forbidden:
		warningf("add/update package: %s", err)
		jsonresponse.Forbidden(w, nil)
		return
	case packages.DomainNotFound:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrDomainNotFound)
		return
	case packages.PackageNotFound:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageNotFound)
		return
	case packages.PackageDomainRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageDomainRequired)
		return
	case packages.PackagePathRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackagePathRequired)
		return
	case packages.PackageVCSRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageVCSRequired)
		return
	case packages.PackageRepoRootRequired:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageRepoRootRequired)
		return
	case packages.PackageAlreadyExists:
		warningf("add/update package: %s", err)
		jsonresponse.BadRequest(w, api.ErrPackageAlreadyExists)
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
	auditf(r, request, action, "%s %s (domain: %s)", p.ID, p.ImportPrefix(), p.Domain.ID)

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func deletePackageAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	id := mux.Vars(r)["id"]

	// Delete package checks permissions.
	p, err := srv.PackagesService.DeletePackage(id, u.ID)
	switch err {
	case packages.Forbidden:
		srv.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.Forbidden(w, nil)
		return
	case packages.DomainNotFound:
		srv.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.BadRequest(w, api.ErrDomainNotFound)
		return
	case packages.PackageNotFound:
		srv.logger.Warningf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.BadRequest(w, api.ErrPackageNotFound)
		return
	case nil:
	default:
		srv.logger.Errorf("package delete api: user %s: delete package %s: %s", u.ID, id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	srv.logger.Debugf("package delete api: %s deleted by %s", p.ID, u.ID)

	auditf(r, nil, "package delete", "%s: %s", p.ID, p.ImportPrefix)

	jsonresponse.OK(w, packagesPackageToAPIPackage(*p, nil))
}

func domainPackagesAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
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

	pkgs, err := srv.PackagesService.PackagesByDomain(id, start, limit)
	if err != nil {
		if err == packages.DomainNotFound {
			srv.logger.Warningf("domain packages api: packages by domain %s: %s", id, err)
			jsonresponse.BadRequest(w, api.ErrDomainNotFound)
			return
		}
		srv.logger.Errorf("domain packages api: packages by domain %s: %s", id, err)
		jsonresponse.InternalServerError(w, nil)
		return
	}

	token := ""
	authorized := false
	for {
		response, err := srv.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist {
				srv.logger.Warningf("domain packages api: user domains %s: %s", u.ID, err)
				break
			}
			if err == packages.DomainNotFound {
				srv.logger.Warningf("domain packages api: user domains %s: %s", u.ID, err)
				break
			}
			srv.logger.Errorf("domain packages api: user domains %s: %s", u.ID, err)
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
		srv.logger.Errorf("domain packages api: domain %s: not allowed for user %s", id, u.ID)
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
