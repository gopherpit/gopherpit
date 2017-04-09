// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/user"
)

func TestPackagesAPI(t *testing.T) {
	if err := startTestServer(nil); err != nil {
		t.Fatal(err)
	}
	defer stopTestServer()

	httpClients := map[string]*api.Client{}
	users := map[string]*user.User{}
	for _, username := range []string{"alice", "bob", "chuck"} {
		t.Run(fmt.Sprintf("create account %s", username), func(t *testing.T) {
			email := username + "@localhost.loc"
			name := strings.ToUpper(username)
			u, err := srv.UserService.CreateUser(&user.Options{
				Email:    &email,
				Username: &username,
				Name:     &name,
			})
			if err != nil {
				t.Fatalf("create user: %s", err)
			}

			users[username] = u

			_, ipV4Net, err := net.ParseCIDR("0.0.0.0/0")
			if err != nil {
				t.Fatalf("parse IPv4 net: %s", err)
			}
			_, ipV6Net, err := net.ParseCIDR("::/0")
			if err != nil {
				t.Fatalf("parse IPv6 net: %s", err)
			}

			k, err := srv.KeyService.CreateKey(u.ID, &key.Options{
				AuthorizedNetworks: &[]net.IPNet{
					*ipV4Net,
					*ipV6Net,
				},
			})
			if err != nil {
				t.Fatalf("create key: %s", err)
			}

			httpClients[username] = api.NewClientWithEndpoint(
				"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
				k.Secret,
			)
			httpClients[username].UserAgent = username + "-gopherpit-test-client"
		})
	}

	domains := map[string][]api.Domain{}

	t.Run("add domain", func(t *testing.T) {
		for username, fqdns := range map[string][]string{
			"alice": {"alice.trusted.com", "alice2.trusted.com", "alice3.trusted.com"},
			"chuck": {"chuck.trusted.com"},
		} {
			for _, fqdn := range fqdns {
				domain, err := httpClients[username].AddDomain(&api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				domains[username] = append(domains[username], domain)
			}
		}
	})

	True := true

	var referencePkgID string

	t.Run("add package", func(t *testing.T) {
		path := "/docker"
		vcs := api.VCSGit
		repoRoot := "https://github.com/docker/docker.git"
		refType := api.RefTypeBranch
		refName := "feature/alice"
		goSource := "alice.trusted.com/docker https://github.com/docker/docker/ https://github.com/docker/docker/tree/master{/dir} https://github.com/docker/docker/blob/master{/dir}/{file}#L{line}"
		redirectURL := "https://godoc.org/alice.trusted.com/docker"
		var err error
		domainID := domains["alice"][0].ID
		pkg, err := httpClients["alice"].AddPackage(&api.PackageOptions{
			Domain:      &domainID,
			Path:        &path,
			VCS:         &vcs,
			RepoRoot:    &repoRoot,
			RefType:     &refType,
			RefName:     &refName,
			GoSource:    &goSource,
			RedirectURL: &redirectURL,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, pkg.DomainID)
		}
		if pkg.Path != path {
			t.Errorf("expected %q, got %q", path, pkg.Path)
		}
		if pkg.VCS != vcs {
			t.Errorf("expected %q, got %q", vcs, pkg.VCS)
		}
		if pkg.RepoRoot != repoRoot {
			t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
		}
		if pkg.RefType != refType {
			t.Errorf("expected %q, got %q", refType, pkg.RefType)
		}
		if pkg.RefName != refName {
			t.Errorf("expected %q, got %q", refName, pkg.RefName)
		}
		if pkg.GoSource != goSource {
			t.Errorf("expected %q, got %q", goSource, pkg.GoSource)
		}
		if pkg.RedirectURL != redirectURL {
			t.Errorf("expected %q, got %q", redirectURL, pkg.RedirectURL)
		}
		if pkg.Disabled != false {
			t.Errorf("expected %v, got %v", false, pkg.Disabled)
		}

		// set update pkgID
		referencePkgID = pkg.ID

		path = "/gopherpit"
		vcs = api.VCSGit
		repoRoot = "https://github.com/gopherpit/gopherpit.git"
		domainID = "alice.trusted.com"
		pkg, err = httpClients["alice"].AddPackage(&api.PackageOptions{
			Domain:   &domainID,
			Path:     &path,
			VCS:      &vcs,
			RepoRoot: &repoRoot,
			Disabled: &True,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, pkg.DomainID)
		}
		if pkg.Path != path {
			t.Errorf("expected %q, got %q", path, pkg.Path)
		}
		if pkg.VCS != vcs {
			t.Errorf("expected %q, got %q", vcs, pkg.VCS)
		}
		if pkg.RepoRoot != repoRoot {
			t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
		}
		if pkg.RefType != "" {
			t.Errorf("expected %q, got %q", "", pkg.RefType)
		}
		if pkg.RefName != "" {
			t.Errorf("expected %q, got %q", "", pkg.RefName)
		}
		if pkg.GoSource != "" {
			t.Errorf("expected %q, got %q", "", pkg.GoSource)
		}
		if pkg.RedirectURL != "" {
			t.Errorf("expected %q, got %q", "", pkg.RedirectURL)
		}
		if pkg.Disabled != true {
			t.Errorf("expected %v, got %v", true, pkg.Disabled)
		}
		t.Run("vcs schemes", func(t *testing.T) {
			for vcs, schemes := range map[api.VCS][]string{
				api.VCSGit:        {"https", "http", "git", "git+ssh", "ssh"},
				api.VCSMercurial:  {"https", "http", "ssh"},
				api.VCSBazaar:     {"https", "http", "bzr", "bzr+ssh"},
				api.VCSSubversion: {"https", "http", "svn", "svn+ssh"},
			} {
				for _, scheme := range schemes {
					path := "/test" + string(vcs) + scheme
					repoRoot := scheme + "://github.com/gopherpit/gopherpit.git"
					domainID := domains["alice"][0].ID
					pkg, err := httpClients["alice"].AddPackage(&api.PackageOptions{
						Domain:   &domainID,
						Path:     &path,
						VCS:      &vcs,
						RepoRoot: &repoRoot,
						Disabled: &True,
					})
					if err != nil {
						t.Errorf("vcs %q, scheme %q: %s", vcs, scheme, err)
					}
					if pkg.Path != path {
						t.Errorf("expected %q, got %q", path, pkg.Path)
					}
					if pkg.VCS != vcs {
						t.Errorf("expected %q, got %q", vcs, pkg.VCS)
					}
					if pkg.RepoRoot != repoRoot {
						t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
					}
				}
			}
		})
		t.Run("reference change", func(t *testing.T) {
			refName := "test"
			path := "test-ref-branch"
			p, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &path,
				VCS:      &api.VCSGit,
				RepoRoot: &pkg.RepoRoot,
				RefType:  &api.RefTypeBranch,
				RefName:  &refName,
			})
			if err != nil {
				t.Error(err)
			}
			if p.RefType != api.RefTypeBranch {
				t.Errorf("expected %q, got %q", api.RefTypeBranch, p.RefType)
			}
			path = "test-ref-tag"
			p, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &path,
				VCS:      &api.VCSGit,
				RepoRoot: &pkg.RepoRoot,
				RefType:  &api.RefTypeTag,
				RefName:  &refName,
			})
			if err != nil {
				t.Error(err)
			}
			if p.RefType != api.RefTypeTag {
				t.Errorf("expected %q, got %q", api.RefTypeTag, p.RefType)
			}
		})
		t.Run("forbidden", func(t *testing.T) {
			path := "/gopherpit"
			vcs := api.VCSGit
			repoRoot := "https://github.com/gopherpit/gopherpit.git"
			domainID := domains["alice"][0].ID
			_, err := httpClients["chuck"].AddPackage(&api.PackageOptions{
				Domain:   &domainID,
				Path:     &path,
				VCS:      &vcs,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}

			domainID = "alice.trusted.com"
			_, err = httpClients["chuck"].AddPackage(&api.PackageOptions{
				Domain:   &domainID,
				Path:     &path,
				VCS:      &vcs,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			path := "/gopherpit"
			vcs := api.VCSGit
			repoRoot := "https://github.com/gopherpit/gopherpit.git"
			domainID := "missing.trusted.com"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &domainID,
				Path:     &path,
				VCS:      &vcs,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
		t.Run("package already exists", func(t *testing.T) {
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackageAlreadyExists {
				t.Errorf("expected %q, got %q", api.ErrPackageAlreadyExists, err)
			}
		})
		t.Run("package domain required", func(t *testing.T) {
			domain := ""
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &domain,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackageDomainRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageDomainRequired, err)
			}
			_, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   nil,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackageDomainRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageDomainRequired, err)
			}
		})
		t.Run("package path required", func(t *testing.T) {
			path := ""
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     nil,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackagePathRequired {
				t.Errorf("expected %q, got %q", api.ErrPackagePathRequired, err)
			}
			_, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackagePathRequired {
				t.Errorf("expected %q, got %q", api.ErrPackagePathRequired, err)
			}
		})
		t.Run("package vcs required", func(t *testing.T) {
			vcs := api.VCS("")
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &vcs,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackageVCSRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageVCSRequired, err)
			}
			_, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      nil,
				RepoRoot: &pkg.RepoRoot,
			})
			if err != api.ErrPackageVCSRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageVCSRequired, err)
			}
		})
		t.Run("package repository root required", func(t *testing.T) {
			repoRoot := ""
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootRequired, err)
			}
			_, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: nil,
			})
			if err != api.ErrPackageRepoRootRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootRequired, err)
			}
		})
		t.Run("package repository root invalid", func(t *testing.T) {
			repoRoot := "::"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootInvalid, err)
			}
		})
		t.Run("package repository root scheme required", func(t *testing.T) {
			repoRoot := "domain/path"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootSchemeRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootSchemeRequired, err)
			}
		})
		t.Run("package repository root scheme invalid", func(t *testing.T) {
			repoRoot := "invalid://domain/path"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootSchemeInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootSchemeInvalid, err)
			}
		})
		t.Run("package repository root host invalid", func(t *testing.T) {
			repoRoot := "https://domain:port/path"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootHostInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootHostInvalid, err)
			}
		})
		t.Run("package reference type invalid", func(t *testing.T) {
			refType := api.RefType("invalid")
			refName := "test"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
				RefType:  &refType,
				RefName:  &refName,
			})
			if err != api.ErrPackageRefTypeInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRefTypeInvalid, err)
			}
		})
		t.Run("package reference name required", func(t *testing.T) {
			refType := api.RefTypeBranch
			refName := ""
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
				RefType:  &refType,
				RefName:  &refName,
			})
			if err != api.ErrPackageRefNameRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRefNameRequired, err)
			}
			_, err = httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &pkg.DomainID,
				Path:     &pkg.Path,
				VCS:      &pkg.VCS,
				RepoRoot: &pkg.RepoRoot,
				RefType:  &refType,
				RefName:  nil,
			})
			if err != api.ErrPackageRefNameRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRefNameRequired, err)
			}
		})
		t.Run("package reference change rejected", func(t *testing.T) {
			for _, vcs := range []api.VCS{api.VCSBazaar, api.VCSMercurial, api.VCSSubversion} {
				refName := "test"
				_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
					Domain:   &pkg.DomainID,
					Path:     &pkg.Path,
					VCS:      &vcs,
					RepoRoot: &pkg.RepoRoot,
					RefType:  &api.RefTypeBranch,
					RefName:  &refName,
				})
				if err != api.ErrPackageRefChangeRejected {
					t.Errorf("%s: expected %q, got %q", vcs, api.ErrPackageRefChangeRejected, err)
				}
			}
		})
		t.Run("package redirect url invalid", func(t *testing.T) {
			redirectURL := "::"
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:      &pkg.DomainID,
				Path:        &pkg.Path,
				VCS:         &pkg.VCS,
				RepoRoot:    &pkg.RepoRoot,
				RedirectURL: &redirectURL,
			})
			if err != api.ErrPackageRedirectURLInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRedirectURLInvalid, err)
			}
		})
	})
	t.Run("update package", func(t *testing.T) {
		path := "/docker-updated"
		vcs := api.VCSGit
		repoRoot := "https://github.com/docker/docker-updated.git"
		refType := api.RefTypeTag
		refName := "feature/alice-updated"
		goSource := "alice.trusted.com/docker-updated https://github.com/docker/docker-updated/ https://github.com/docker/docker-updated/tree/master{/dir} https://github.com/docker/docker-updated/blob/master{/dir}/{file}#L{line}"
		redirectURL := "https://godoc.org/alice.trusted.com/docker-updated"
		var err error
		domainID := domains["alice"][0].ID
		pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
			Domain:      &domainID,
			Path:        &path,
			VCS:         &vcs,
			RepoRoot:    &repoRoot,
			RefType:     &refType,
			RefName:     &refName,
			GoSource:    &goSource,
			RedirectURL: &redirectURL,
			Disabled:    &True,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, pkg.DomainID)
		}
		if pkg.Path != path {
			t.Errorf("expected %q, got %q", path, pkg.Path)
		}
		if pkg.VCS != vcs {
			t.Errorf("expected %q, got %q", vcs, pkg.VCS)
		}
		if pkg.RepoRoot != repoRoot {
			t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
		}
		if pkg.RefType != refType {
			t.Errorf("expected %q, got %q", refType, pkg.RefType)
		}
		if pkg.RefName != refName {
			t.Errorf("expected %q, got %q", refName, pkg.RefName)
		}
		if pkg.GoSource != goSource {
			t.Errorf("expected %q, got %q", goSource, pkg.GoSource)
		}
		if pkg.RedirectURL != redirectURL {
			t.Errorf("expected %q, got %q", redirectURL, pkg.RedirectURL)
		}
		if pkg.Disabled != true {
			t.Errorf("expected %v, got %v", true, pkg.Disabled)
		}
		refType = ""
		refName = ""
		pkg, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
			VCS:     &api.VCSSubversion,
			RefType: &refType,
			RefName: &refName,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.RefType != refType {
			t.Errorf("expected %q, got %q", refType, pkg.RefType)
		}
		if pkg.RefName != refName {
			t.Errorf("expected %q, got %q", refName, pkg.RefName)
		}
		t.Run("domain", func(t *testing.T) {
			domainID := domains["alice"][1].ID
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domainID,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.DomainID != domains["alice"][1].ID {
				t.Errorf("expected %q, got %q", domains["alice"][1].ID, pkg.DomainID)
			}

			domainID = "alice3.trusted.com"
			pkg, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domainID,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.DomainID != domains["alice"][2].ID {
				t.Errorf("expected %q, got %q", domains["alice"][2].ID, pkg.DomainID)
			}
		})
		t.Run("path", func(t *testing.T) {
			path := "/some-new-path"
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Path: &path,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.Path != path {
				t.Errorf("expected %q, got %q", path, pkg.Path)
			}
		})
		t.Run("vcs", func(t *testing.T) {
			for _, vcs := range []api.VCS{api.VCSBazaar, api.VCSGit, api.VCSMercurial, api.VCSSubversion} {
				pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
					VCS: &vcs,
				})
				if err != nil {
					t.Fatal(err)
				}
				if pkg.VCS != vcs {
					t.Errorf("expected %q, got %q", vcs, pkg.VCS)
				}
			}
		})
		t.Run("vcs schemes", func(t *testing.T) {
			schemes := map[api.VCS][]string{
				api.VCSGit:        {"https", "http", "git", "git+ssh", "ssh"},
				api.VCSMercurial:  {"https", "http", "ssh"},
				api.VCSBazaar:     {"https", "http", "bzr", "bzr+ssh"},
				api.VCSSubversion: {"https", "http", "svn", "svn+ssh"},
			}
			for _, vcs := range []api.VCS{api.VCSGit, api.VCSBazaar, api.VCSMercurial, api.VCSSubversion} {
				for _, scheme := range schemes[vcs] {
					path := "/test-update" + string(vcs) + scheme
					repoRoot := scheme + "://github.com/gopherpit/gopherpit.git"
					domainID := domains["alice"][0].ID
					pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
						Domain:   &domainID,
						Path:     &path,
						VCS:      &vcs,
						RepoRoot: &repoRoot,
						Disabled: &True,
					})
					if err != nil {
						t.Errorf("vcs %q, scheme %q: %s", vcs, scheme, err)
					}
					if pkg.Path != path {
						t.Errorf("expected %q, got %q", path, pkg.Path)
					}
					if pkg.VCS != vcs {
						t.Errorf("expected %q, got %q", vcs, pkg.VCS)
					}
					if pkg.RepoRoot != repoRoot {
						t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
					}
				}
			}
		})
		t.Run("repo root", func(t *testing.T) {
			repoRoot := "https://github.com/me/my-application.git"
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.RepoRoot != repoRoot {
				t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
			}
		})
		t.Run("reference change", func(t *testing.T) {
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				VCS: &api.VCSGit,
			})
			if err != nil {
				t.Error(err)
			}
			refName := "test-update"
			p, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &api.RefTypeBranch,
				RefName: &refName,
			})
			if err != nil {
				t.Error(err)
			}
			if p.RefType != api.RefTypeBranch {
				t.Errorf("expected %q, got %q", api.RefTypeBranch, p.RefType)
			}
			p, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &api.RefTypeTag,
				RefName: &refName,
			})
			if err != nil {
				t.Error(err)
			}
			if p.RefType != api.RefTypeTag {
				t.Errorf("expected %q, got %q", api.RefTypeTag, p.RefType)
			}
		})
		t.Run("go source", func(t *testing.T) {
			goSource := "go-source-update"
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				GoSource: &goSource,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.GoSource != goSource {
				t.Errorf("expected %q, got %q", goSource, pkg.GoSource)
			}
		})
		t.Run("redirect url", func(t *testing.T) {
			redirectURL := "https://gopherpit.com"
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RedirectURL: &redirectURL,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.RedirectURL != redirectURL {
				t.Errorf("expected %q, got %q", redirectURL, pkg.RedirectURL)
			}
		})
		t.Run("disabled", func(t *testing.T) {
			disabled := true
			pkg, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Disabled: &disabled,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.Disabled != disabled {
				t.Errorf("expected %v, got %v", disabled, pkg.Disabled)
			}
			disabled = false
			pkg, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Disabled: &disabled,
			})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.Disabled != disabled {
				t.Errorf("expected %v, got %v", disabled, pkg.Disabled)
			}
		})
		t.Run("forbidden", func(t *testing.T) {
			repoRoot := "https://github.com/gopherpit/gopherpit.git"
			_, err := httpClients["chuck"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}

			domainID := domains["chuck"][0].ID
			_, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domainID,
			})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
			domainID = "chuck.trusted.com"
			_, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domainID,
			})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			domainID := "missing.trusted.com"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domainID,
			})
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
		t.Run("package not found", func(t *testing.T) {
			_, err := httpClients["alice"].UpdatePackage(referencePkgID+"1", &api.PackageOptions{})
			if err != api.ErrPackageNotFound {
				t.Errorf("expected %q, got %q", api.ErrPackageNotFound, err)
			}
		})
		t.Run("package already exists", func(t *testing.T) {
			path := "/gopherpit-update-test"
			vcs := api.VCSGit
			repoRoot := "https://github.com/gopherpit/gopherpit.git"
			domainID := domains["alice"][0].ID
			_, err := httpClients["alice"].AddPackage(&api.PackageOptions{
				Domain:   &domainID,
				Path:     &path,
				VCS:      &vcs,
				RepoRoot: &repoRoot,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Path: &path,
			})
			if err != api.ErrPackageAlreadyExists {
				t.Errorf("expected %q, got %q", api.ErrPackageAlreadyExists, err)
			}
		})
		t.Run("package domain required", func(t *testing.T) {
			domain := ""
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Domain: &domain,
			})
			if err != api.ErrPackageDomainRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageDomainRequired, err)
			}
		})
		t.Run("package path required", func(t *testing.T) {
			path := ""
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				Path: &path,
			})
			if err != api.ErrPackagePathRequired {
				t.Errorf("expected %q, got %q", api.ErrPackagePathRequired, err)
			}
		})
		t.Run("package vcs required", func(t *testing.T) {
			vcs := api.VCS("")
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				VCS: &vcs,
			})
			if err != api.ErrPackageVCSRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageVCSRequired, err)
			}
		})
		t.Run("package repository root required", func(t *testing.T) {
			repoRoot := ""
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootRequired, err)
			}
		})
		t.Run("package repository root invalid", func(t *testing.T) {
			repoRoot := "::"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootInvalid, err)
			}
		})
		t.Run("package repository root scheme required", func(t *testing.T) {
			refType := api.RefType("")
			refName := ""
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &refType,
				RefName: &refName,
			})
			if err != nil {
				t.Error(err)
			}

			repoRoot := "domain/path"
			_, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootSchemeRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootSchemeRequired, err)
			}
		})
		t.Run("package repository root scheme invalid", func(t *testing.T) {
			repoRoot := "invalid://domain/path"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootSchemeInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootSchemeInvalid, err)
			}
		})
		t.Run("package repository root host invalid", func(t *testing.T) {
			repoRoot := "https://domain:port/path"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RepoRoot: &repoRoot,
			})
			if err != api.ErrPackageRepoRootHostInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRepoRootHostInvalid, err)
			}
		})
		t.Run("package reference type invalid", func(t *testing.T) {
			refType := api.RefType("invalid")
			refName := "test"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &refType,
				RefName: &refName,
			})
			if err != api.ErrPackageRefTypeInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRefTypeInvalid, err)
			}
		})
		t.Run("package reference name required", func(t *testing.T) {
			refType := api.RefTypeBranch
			refName := ""
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &refType,
				RefName: &refName,
			})
			if err != api.ErrPackageRefNameRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRefNameRequired, err)
			}
			_, err = httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RefType: &refType,
				RefName: nil,
			})
			if err != api.ErrPackageRefNameRequired {
				t.Errorf("expected %q, got %q", api.ErrPackageRefNameRequired, err)
			}
		})
		t.Run("package reference change rejected", func(t *testing.T) {
			for _, vcs := range []api.VCS{api.VCSBazaar, api.VCSMercurial, api.VCSSubversion} {
				refName := "test"
				_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
					VCS:     &vcs,
					RefType: &api.RefTypeBranch,
					RefName: &refName,
				})
				if err != api.ErrPackageRefChangeRejected {
					t.Errorf("%s: expected %q, got %q", vcs, api.ErrPackageRefChangeRejected, err)
				}
			}
		})
		t.Run("package redirect url invalid", func(t *testing.T) {
			redirectURL := "::"
			_, err := httpClients["alice"].UpdatePackage(referencePkgID, &api.PackageOptions{
				RedirectURL: &redirectURL,
			})
			if err != api.ErrPackageRedirectURLInvalid {
				t.Errorf("expected %q, got %q", api.ErrPackageRedirectURLInvalid, err)
			}
		})
	})

	var refPkg api.Package

	t.Run("get package", func(t *testing.T) {
		path := "/docker2"
		vcs := api.VCSGit
		repoRoot := "https://github.com/docker/docker.git"
		refType := api.RefTypeBranch
		refName := "feature/alice"
		goSource := "alice.trusted.com/docker2 https://github.com/docker/docker/ https://github.com/docker/docker/tree/master{/dir} https://github.com/docker/docker/blob/master{/dir}/{file}#L{line}"
		redirectURL := "https://godoc.org/alice.trusted.com/docker2"
		var err error
		domainID := domains["alice"][0].ID
		pkg, err := httpClients["alice"].AddPackage(&api.PackageOptions{
			Domain:      &domainID,
			Path:        &path,
			VCS:         &vcs,
			RepoRoot:    &repoRoot,
			RefType:     &refType,
			RefName:     &refName,
			GoSource:    &goSource,
			RedirectURL: &redirectURL,
			Disabled:    &True,
		})
		if err != nil {
			t.Fatal(err)
		}
		if pkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, pkg.DomainID)
		}
		if pkg.Path != path {
			t.Errorf("expected %q, got %q", path, pkg.Path)
		}
		if pkg.VCS != vcs {
			t.Errorf("expected %q, got %q", vcs, pkg.VCS)
		}
		if pkg.RepoRoot != repoRoot {
			t.Errorf("expected %q, got %q", repoRoot, pkg.RepoRoot)
		}
		if pkg.RefType != refType {
			t.Errorf("expected %q, got %q", refType, pkg.RefType)
		}
		if pkg.RefName != refName {
			t.Errorf("expected %q, got %q", refName, pkg.RefName)
		}
		if pkg.GoSource != goSource {
			t.Errorf("expected %q, got %q", goSource, pkg.GoSource)
		}
		if pkg.RedirectURL != redirectURL {
			t.Errorf("expected %q, got %q", redirectURL, pkg.RedirectURL)
		}
		if pkg.Disabled != true {
			t.Errorf("expected %v, got %v", true, pkg.Disabled)
		}

		refPkg, err = httpClients["alice"].Package(pkg.ID)
		if err != nil {
			t.Fatal(err)
		}
		if refPkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, refPkg.DomainID)
		}
		if refPkg.Path != path {
			t.Errorf("expected %q, got %q", path, refPkg.Path)
		}
		if refPkg.VCS != vcs {
			t.Errorf("expected %q, got %q", vcs, refPkg.VCS)
		}
		if refPkg.RepoRoot != repoRoot {
			t.Errorf("expected %q, got %q", repoRoot, refPkg.RepoRoot)
		}
		if refPkg.RefType != refType {
			t.Errorf("expected %q, got %q", refType, refPkg.RefType)
		}
		if refPkg.RefName != refName {
			t.Errorf("expected %q, got %q", refName, refPkg.RefName)
		}
		if refPkg.GoSource != goSource {
			t.Errorf("expected %q, got %q", goSource, refPkg.GoSource)
		}
		if refPkg.RedirectURL != redirectURL {
			t.Errorf("expected %q, got %q", redirectURL, refPkg.RedirectURL)
		}
		if refPkg.Disabled != True {
			t.Errorf("expected %v, got %v", True, refPkg.Disabled)
		}
	})
	t.Run("delete package", func(t *testing.T) {
		pkg, err := httpClients["alice"].DeletePackage(refPkg.ID)
		if err != nil {
			t.Fatal(err)
		}
		if pkg.DomainID != domains["alice"][0].ID {
			t.Errorf("expected %q, got %q", domains["alice"][0].ID, pkg.DomainID)
		}
		if pkg.Path != refPkg.Path {
			t.Errorf("expected %q, got %q", refPkg.Path, pkg.Path)
		}
		if pkg.VCS != refPkg.VCS {
			t.Errorf("expected %q, got %q", refPkg.VCS, pkg.VCS)
		}
		if pkg.RepoRoot != refPkg.RepoRoot {
			t.Errorf("expected %q, got %q", refPkg.RepoRoot, pkg.RepoRoot)
		}
		if pkg.RefType != refPkg.RefType {
			t.Errorf("expected %q, got %q", refPkg.RefType, pkg.RefType)
		}
		if pkg.RefName != refPkg.RefName {
			t.Errorf("expected %q, got %q", refPkg.RefName, pkg.RefName)
		}
		if pkg.GoSource != refPkg.GoSource {
			t.Errorf("expected %q, got %q", refPkg.GoSource, pkg.GoSource)
		}
		if pkg.RedirectURL != refPkg.RedirectURL {
			t.Errorf("expected %q, got %q", refPkg.RedirectURL, pkg.RedirectURL)
		}
		if pkg.Disabled != true {
			t.Errorf("expected %v, got %v", true, pkg.Disabled)
		}
	})
	t.Run("list domains", func(t *testing.T) {
		want := api.PackagesPage{Packages: []api.Package{api.Package{FQDN: "alice.trusted.com", Path: "/gopherpit", VCS: "git", RepoRoot: "https://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/gopherpit-update-test", VCS: "git", RepoRoot: "https://github.com/gopherpit/gopherpit.git"}, api.Package{FQDN: "alice.trusted.com", Path: "/test-ref-branch", VCS: "git", RepoRoot: "https://github.com/gopherpit/gopherpit.git", RefType: "branch", RefName: "test"}, api.Package{FQDN: "alice.trusted.com", Path: "/test-ref-tag", VCS: "git", RepoRoot: "https://github.com/gopherpit/gopherpit.git", RefType: "tag", RefName: "test"}, api.Package{FQDN: "alice.trusted.com", Path: "/test-updatesvnsvn+ssh", VCS: "git", RepoRoot: "https://github.com/me/my-application.git", GoSource: "go-source-update", RedirectURL: "https://gopherpit.com"}, api.Package{FQDN: "alice.trusted.com", Path: "/testbzrbzr", VCS: "bzr", RepoRoot: "bzr://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testbzrbzr+ssh", VCS: "bzr", RepoRoot: "bzr+ssh://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testbzrhttp", VCS: "bzr", RepoRoot: "http://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testbzrhttps", VCS: "bzr", RepoRoot: "https://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testgitgit", VCS: "git", RepoRoot: "git://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testgitgit+ssh", VCS: "git", RepoRoot: "git+ssh://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testgithttp", VCS: "git", RepoRoot: "http://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testgithttps", VCS: "git", RepoRoot: "https://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testgitssh", VCS: "git", RepoRoot: "ssh://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testhghttp", VCS: "hg", RepoRoot: "http://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testhghttps", VCS: "hg", RepoRoot: "https://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testhgssh", VCS: "hg", RepoRoot: "ssh://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testsvnhttp", VCS: "svn", RepoRoot: "http://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testsvnhttps", VCS: "svn", RepoRoot: "https://github.com/gopherpit/gopherpit.git", Disabled: true}, api.Package{FQDN: "alice.trusted.com", Path: "/testsvnsvn", VCS: "svn", RepoRoot: "svn://github.com/gopherpit/gopherpit.git", Disabled: true}}, Count: 20, Previous: "", Next: "/testsvnsvn+ssh"}

		page, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		if page.Count != want.Count {
			t.Errorf("expected %v, got %v", want.Count, page.Count)
		}
		if page.Next != want.Next {
			t.Errorf("expected %q, got %q", want.Next, page.Next)
		}
		if page.Previous != want.Previous {
			t.Errorf("expected %q, got %q", want.Previous, page.Previous)
		}
		for i := range want.Packages {
			if page.Packages[i].FQDN != want.Packages[i].FQDN {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].FQDN, page.Packages[i].FQDN)
			}
			if page.Packages[i].Path != want.Packages[i].Path {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].Path, page.Packages[i].Path)
			}
			if page.Packages[i].RefType != want.Packages[i].RefType {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].RefType, page.Packages[i].RefType)
			}
			if page.Packages[i].RefName != want.Packages[i].RefName {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].RefName, page.Packages[i].RefName)
			}
			if page.Packages[i].VCS != want.Packages[i].VCS {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].VCS, page.Packages[i].VCS)
			}
			if page.Packages[i].RepoRoot != want.Packages[i].RepoRoot {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].RepoRoot, page.Packages[i].RepoRoot)
			}
			if page.Packages[i].GoSource != want.Packages[i].GoSource {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].GoSource, page.Packages[i].GoSource)
			}
			if page.Packages[i].RedirectURL != want.Packages[i].RedirectURL {
				t.Errorf("%d: expected %q, got %q", i, want.Packages[i].RedirectURL, page.Packages[i].RedirectURL)
			}
			if page.Packages[i].Disabled != want.Packages[i].Disabled {
				t.Errorf("%d: expected %v, got %v", i, want.Packages[i].Disabled, page.Packages[i].Disabled)
			}
		}
		t.Run("pagination limit 6", func(t *testing.T) {
			page1, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, "", 6)
			if err != nil {
				t.Fatal(err)
			}
			if page1.Count != 6 {
				t.Errorf("expected %v, got %v", 6, page1.Count)
			}
			if len(page1.Packages) != 6 {
				t.Errorf("expected %v, got %v", 6, len(page1.Packages))
			}
			page2, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, page1.Next, 6)
			if err != nil {
				t.Fatal(err)
			}
			if page2.Count != 6 {
				t.Errorf("expected %v, got %v", 6, page2.Count)
			}
			if len(page2.Packages) != 6 {
				t.Errorf("expected %v, got %v", 6, len(page2.Packages))
			}
			page3, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, page2.Next, 6)
			if err != nil {
				t.Fatal(err)
			}
			if page3.Count != 6 {
				t.Errorf("expected %v, got %v", 6, page3.Count)
			}
			if len(page3.Packages) != 6 {
				t.Errorf("expected %v, got %v", 6, len(page3.Packages))
			}
			if page3.Previous != page1.Next {
				t.Errorf("expected %q, got %q", page1.Next, page3.Previous)
			}
			page4, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, page3.Next, 6)
			if err != nil {
				t.Fatal(err)
			}
			if page4.Count != 3 {
				t.Errorf("expected %v, got %v", 3, page4.Count)
			}
			if len(page4.Packages) != 3 {
				t.Errorf("expected %v, got %v", 3, len(page4.Packages))
			}
			if page4.Previous != page2.Next {
				t.Errorf("expected %q, got %q", page2.Next, page4.Previous)
			}
		})
		t.Run("forbidden", func(t *testing.T) {
			_, err := httpClients["chuck"].DomainPackages(domains["alice"][0].ID, "", 0)
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			_, err := httpClients["alice"].DomainPackages("missing.example.com", "", 0)
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
		t.Run("package not found", func(t *testing.T) {
			_, err := httpClients["alice"].DomainPackages(domains["alice"][0].ID, "/missing-package", 0)
			if err != api.ErrPackageNotFound {
				t.Errorf("expected %q, got %q", api.ErrPackageNotFound, err)
			}
		})
	})
}
