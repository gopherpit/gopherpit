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
	"gopherpit.com/gopherpit/client"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/user"
)

func TestDomainsAPI(t *testing.T) {
	if err := startTestServer(nil); err != nil {
		t.Fatal(err)
	}
	defer stopTestServer()

	httpClients := map[string]*client.Client{}
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

			httpClients[username] = client.NewClientWithEndpoint(
				"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
				k.Secret,
			)
			httpClients[username].UserAgent = username + "-gopherpit-test-client"
		})
	}

	True := true
	domains := map[string][]string{
		"alice": {
			"alice.trusted.com",
			"gopherpit.localhost",
			"localhost",
			"to-delete.localhost",
			"to-rename.localhost",
			"trusted.com",
		},
		"bob": {
			"2.trusted.com",
			"bob.localhost",
			"bob.trusted.com",
		},
		"chuck": {
			"chuck.localhost",
		},
	}

	t.Run("add domain", func(t *testing.T) {
		for username, fqdns := range domains {
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
			}
		}
		t.Run("disabled domain", func(t *testing.T) {
			fqdn := "disabled.localhost"
			o := &api.DomainOptions{
				FQDN:     &fqdn,
				Disabled: &True,
			}
			domain, err := httpClients["alice"].AddDomain(o)
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
			if domain.Disabled != true {
				t.Errorf("expected %v, got %q", true, domain.Disabled)
			}
		})
		t.Run("with different owner", func(t *testing.T) {
			fqdn := "to-bob.localhost"
			o := &api.DomainOptions{
				FQDN:        &fqdn,
				OwnerUserID: &users["bob"].ID,
			}
			domain, err := httpClients["alice"].AddDomain(o)
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
			if domain.OwnerUserID != users["bob"].ID {
				t.Errorf("expected %q, got %q", users["bob"].ID, domain.OwnerUserID)
			}
		})
		t.Run("with certificate ignore", func(t *testing.T) {
			fqdn := "cert-ignore.localhost"
			o := &api.DomainOptions{
				FQDN:              &fqdn,
				CertificateIgnore: &True,
			}
			domain, err := httpClients["alice"].AddDomain(o)
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
			if domain.CertificateIgnore != true {
				t.Errorf("expected %v, got %q", true, domain.CertificateIgnore)
			}
		})
		t.Run("domain already exists", func(t *testing.T) {
			for _, fqdn := range []string{
				"localhost",
				"bob.localhost",
			} {
				_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainAlreadyExists {
					t.Errorf("expected %q, got %q", api.ErrDomainAlreadyExists, err)
				}
			}
		})
		t.Run("domain fqdn required", func(t *testing.T) {
			_, err := httpClients["alice"].AddDomain(&api.DomainOptions{})
			if err != api.ErrDomainFQDNRequired {
				t.Errorf("expected %q, got %q", api.ErrDomainFQDNRequired, err)
			}
			fqdn := ""
			_, err = httpClients["alice"].AddDomain(&api.DomainOptions{
				FQDN: &fqdn,
			})
			if err != api.ErrDomainFQDNRequired {
				t.Errorf("expected %q, got %q", api.ErrDomainFQDNRequired, err)
			}
		})
		t.Run("domain fqdn invalid", func(t *testing.T) {
			for _, fqdn := range []string{
				"domain.gopherpit",
				".",
			} {
				_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainFQDNInvalid {
					t.Errorf("expected %q, got %q", api.ErrDomainFQDNInvalid, err)
				}
			}
		})
		t.Run("domain not available", func(t *testing.T) {
			for _, fqdn := range []string{
				"forbidden.com",
				"x.forbidden.com",
				"y.x.forbidden.com",
			} {
				_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainNotAvailable {
					t.Errorf("expected %q, got %q", api.ErrDomainNotAvailable, err)
				}
			}
		})
		t.Run("domain with too many subdomains", func(t *testing.T) {
			for _, fqdn := range []string{
				"x.y.localhost",
				"x.y.z.localhost",
			} {
				_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainWithTooManySubdomains {
					t.Errorf("expected %q, got %q", api.ErrDomainWithTooManySubdomains, err)
				}
			}
		})
		t.Run("domain needs verification", func(t *testing.T) {
			fqdn := "example.com"
			_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
				FQDN: &fqdn,
			})
			if err != api.ErrDomainNeedsVerification {
				t.Errorf("expected %q, got %q", api.ErrDomainNeedsVerification, err)
			}
		})
		t.Run("user does not exist", func(t *testing.T) {
			fqdn := "john.localhost"
			owner := "john"
			_, err := httpClients["alice"].AddDomain(&api.DomainOptions{
				FQDN:        &fqdn,
				OwnerUserID: &owner,
			})
			if err != api.ErrUserDoesNotExist {
				t.Errorf("expected %q, got %q", api.ErrUserDoesNotExist, err)
			}
		})
	})

	t.Run("get domain", func(t *testing.T) {
		for username, fqdns := range domains {
			for _, fqdn := range fqdns {
				domain, err := httpClients[username].Domain(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
			}
		}
		t.Run("forbidden", func(t *testing.T) {
			_, err := httpClients["chuck"].Domain("localhost")
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			_, err := httpClients["alice"].Domain("unknown.localhost")
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
	})

	t.Run("update domain", func(t *testing.T) {
		t.Run("rename to-rename.localhost", func(t *testing.T) {
			fqdn := "alice.localhost"
			domain, err := httpClients["alice"].UpdateDomain("to-rename.localhost", &api.DomainOptions{
				FQDN: &fqdn,
			})
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
		})
		t.Run("disable localhost", func(t *testing.T) {
			fqdn := "localhost"
			domain, err := httpClients["alice"].UpdateDomain(fqdn, &api.DomainOptions{
				Disabled: &True,
			})
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
			if domain.Disabled != true {
				t.Errorf("expected %v, got %q", true, domain.Disabled)
			}
		})
		t.Run("ignore certificate localhost", func(t *testing.T) {
			fqdn := "localhost"
			domain, err := httpClients["alice"].UpdateDomain(fqdn, &api.DomainOptions{
				CertificateIgnore: &True,
			})
			if err != nil {
				t.Fatal(err)
			}
			if domain.FQDN != fqdn {
				t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
			}
			if domain.CertificateIgnore != true {
				t.Errorf("expected %v, got %q", true, domain.CertificateIgnore)
			}
		})
		t.Run("grant and revoke user", func(t *testing.T) {
			fqdn := "gopherpit.localhost"
			user := users["bob"]
			t.Run("grant by id", func(t *testing.T) {
				err := httpClients["alice"].GrantDomainUser(fqdn, user.ID)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{user.ID}) {
					t.Errorf("expected %q, got %q", []string{user.ID}, users.UserIDs)
				}
			})
			t.Run("revoke by id", func(t *testing.T) {
				err := httpClients["alice"].RevokeDomainUser(fqdn, user.ID)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{}) {
					t.Errorf("expected %q, got %q", []string{}, users.UserIDs)
				}
			})
			t.Run("grant by username", func(t *testing.T) {
				err := httpClients["alice"].GrantDomainUser(fqdn, user.Username)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{user.ID}) {
					t.Errorf("expected %q, got %q", []string{user.ID}, users.UserIDs)
				}
			})
			t.Run("revoke by username", func(t *testing.T) {
				err := httpClients["alice"].RevokeDomainUser(fqdn, user.Username)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{}) {
					t.Errorf("expected %q, got %q", []string{}, users.UserIDs)
				}
			})
			t.Run("grant by email", func(t *testing.T) {
				err := httpClients["alice"].GrantDomainUser(fqdn, user.Email)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{user.ID}) {
					t.Errorf("expected %q, got %q", []string{user.ID}, users.UserIDs)
				}
			})
			t.Run("revoke by email", func(t *testing.T) {
				err := httpClients["alice"].RevokeDomainUser(fqdn, user.Email)
				if err != nil {
					t.Fatal(err)
				}

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{}) {
					t.Errorf("expected %q, got %q", []string{}, users.UserIDs)
				}
			})
			t.Run("grant and list", func(t *testing.T) {
				err := httpClients["alice"].GrantDomainUser(fqdn, user.ID)
				if err != nil {
					t.Fatal(err)
				}

				ownerID := users["alice"].ID

				users, err := httpClients["alice"].DomainUsers(fqdn)
				if err != nil {
					t.Fatal(err)
				}
				if !stringSliceEqual(users.UserIDs, []string{user.ID}) {
					t.Errorf("expected %q, got %q", []string{user.ID}, users.UserIDs)
				}
				if users.OwnerUserID != ownerID {
					t.Errorf("expected %q, got %q", ownerID, users.OwnerUserID)
				}
			})
		})
		t.Run("grant domain user", func(t *testing.T) {
			fqdn := "gopherpit.localhost"
			user := users["bob"]
			t.Run("forbidden", func(t *testing.T) {
				t.Parallel()

				err := httpClients["chuck"].GrantDomainUser(fqdn, user.ID)
				if err != api.ErrForbidden {
					t.Errorf("expected %q, got %q", api.ErrForbidden, err)
				}
			})
			t.Run("domain not found", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].GrantDomainUser("unknown.localhost", user.ID)
				if err != api.ErrDomainNotFound {
					t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
				}
			})
			t.Run("user does not exist", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].GrantDomainUser(fqdn, "john")
				if err != api.ErrUserDoesNotExist {
					t.Errorf("expected %q, got %q", api.ErrUserDoesNotExist, err)
				}
			})
			t.Run("user already granted", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].GrantDomainUser(fqdn, user.ID)
				if err != api.ErrUserAlreadyGranted {
					t.Errorf("expected %q, got %q", api.ErrUserAlreadyGranted, err)
				}
			})
		})
		t.Run("revoke domain user", func(t *testing.T) {
			fqdn := "gopherpit.localhost"
			user := users["bob"]
			t.Run("forbidden", func(t *testing.T) {
				t.Parallel()

				err := httpClients["chuck"].RevokeDomainUser(fqdn, user.ID)
				if err != api.ErrForbidden {
					t.Errorf("expected %q, got %q", api.ErrForbidden, err)
				}
			})
			t.Run("domain not found", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].RevokeDomainUser("unknown.localhost", user.ID)
				if err != api.ErrDomainNotFound {
					t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
				}
			})
			t.Run("user does not exist", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].RevokeDomainUser(fqdn, "john")
				if err != api.ErrUserDoesNotExist {
					t.Errorf("expected %q, got %q", api.ErrUserDoesNotExist, err)
				}
			})
			t.Run("user not granted", func(t *testing.T) {
				t.Parallel()

				err := httpClients["alice"].RevokeDomainUser(fqdn, "chuck")
				if err != api.ErrUserNotGranted {
					t.Errorf("expected %q, got %q", api.ErrUserNotGranted, err)
				}
			})
		})
		t.Run("list domain users", func(t *testing.T) {
			fqdn := "gopherpit.localhost"
			t.Run("forbidden", func(t *testing.T) {
				t.Parallel()

				_, err := httpClients["chuck"].DomainUsers(fqdn)
				if err != api.ErrForbidden {
					t.Errorf("expected %q, got %q", api.ErrForbidden, err)
				}
			})
			t.Run("domain not found", func(t *testing.T) {
				t.Parallel()

				_, err := httpClients["alice"].DomainUsers("unknown.localhost")
				if err != api.ErrDomainNotFound {
					t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
				}
			})
		})
		t.Run("change owner localhost", func(t *testing.T) {
			t.Run("by id", func(t *testing.T) {
				fqdn := "localhost"
				owner := users["bob"]
				domain, err := httpClients["alice"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.ID,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}

				owner = users["alice"]
				domain, err = httpClients["bob"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.ID,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}
			})
			t.Run("by username", func(t *testing.T) {
				fqdn := "localhost"
				owner := users["bob"]
				domain, err := httpClients["alice"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.Username,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}

				owner = users["alice"]
				domain, err = httpClients["bob"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.Username,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}
			})
			t.Run("by email", func(t *testing.T) {
				fqdn := "localhost"
				owner := users["bob"]
				domain, err := httpClients["alice"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.Email,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}

				owner = users["alice"]
				domain, err = httpClients["bob"].UpdateDomain(fqdn, &api.DomainOptions{
					OwnerUserID: &owner.Email,
				})
				if err != nil {
					t.Fatal(err)
				}
				if domain.FQDN != fqdn {
					t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
				}
				if domain.OwnerUserID != owner.ID {
					t.Errorf("expected %q, got %q", owner.ID, domain.OwnerUserID)
				}
			})
		})
		t.Run("forbidden", func(t *testing.T) {
			_, err := httpClients["chuck"].UpdateDomain("localhost", &api.DomainOptions{})
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			_, err := httpClients["alice"].UpdateDomain("example.com", &api.DomainOptions{})
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
		t.Run("domain already exists", func(t *testing.T) {
			for _, fqdn := range []string{
				"localhost",
				"bob.localhost",
			} {
				_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainAlreadyExists {
					t.Errorf("expected %q, got %q", api.ErrDomainAlreadyExists, err)
				}
			}
		})
		t.Run("domain fqdn invalid", func(t *testing.T) {
			for _, fqdn := range []string{
				"domain.gopherpit",
				".",
			} {
				_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainFQDNInvalid {
					t.Errorf("expected %q, got %q", api.ErrDomainFQDNInvalid, err)
				}
			}
		})
		t.Run("domain not available", func(t *testing.T) {
			for _, fqdn := range []string{
				"forbidden.com",
				"x.forbidden.com",
				"y.x.forbidden.com",
			} {
				_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainNotAvailable {
					t.Errorf("expected %q, got %q", api.ErrDomainNotAvailable, err)
				}
			}
		})
		t.Run("domain with too many subdomains", func(t *testing.T) {
			for _, fqdn := range []string{
				"x.y.localhost",
				"x.y.z.localhost",
			} {
				_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
					FQDN: &fqdn,
				})
				if err != api.ErrDomainWithTooManySubdomains {
					t.Errorf("expected %q, got %q", api.ErrDomainWithTooManySubdomains, err)
				}
			}
		})
		t.Run("domain needs verification", func(t *testing.T) {
			fqdn := "example.com"
			_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
				FQDN: &fqdn,
			})
			if err != api.ErrDomainNeedsVerification {
				t.Errorf("expected %q, got %q", api.ErrDomainNeedsVerification, err)
			}
		})
		t.Run("user does not exist", func(t *testing.T) {
			fqdn := "john.localhost"
			owner := "john"
			_, err := httpClients["alice"].UpdateDomain("gopherpit.localhost", &api.DomainOptions{
				FQDN:        &fqdn,
				OwnerUserID: &owner,
			})
			if err != api.ErrUserDoesNotExist {
				t.Errorf("expected %q, got %q", api.ErrUserDoesNotExist, err)
			}
		})
	})
	t.Run("delete domain", func(t *testing.T) {
		fqdn := "to-delete.localhost"
		domain, err := httpClients["alice"].DeleteDomain(fqdn)
		if err != nil {
			t.Fatal(err)
		}
		if domain.FQDN != fqdn {
			t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
		}

		domain, err = httpClients["alice"].Domain(fqdn)
		if err != api.ErrDomainNotFound {
			t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
		}
		t.Run("forbidden", func(t *testing.T) {
			_, err := httpClients["chuck"].DeleteDomain("localhost")
			if err != api.ErrForbidden {
				t.Errorf("expected %q, got %q", api.ErrForbidden, err)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			_, err := httpClients["alice"].DeleteDomain("unknown.localhost")
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
	})
	t.Run("list domains", func(t *testing.T) {
		t.Run("by users", func(t *testing.T) {
			expected := map[string][]api.Domain{
				"alice": {api.Domain{FQDN: "alice.localhost"}, api.Domain{FQDN: "alice.trusted.com"}, api.Domain{FQDN: "cert-ignore.localhost", CertificateIgnore: true}, api.Domain{FQDN: "disabled.localhost", Disabled: true}, api.Domain{FQDN: "gopherpit.localhost"}, api.Domain{FQDN: "localhost", CertificateIgnore: true, Disabled: true}, api.Domain{FQDN: "trusted.com"}},
				"bob":   {api.Domain{FQDN: "2.trusted.com"}, api.Domain{FQDN: "bob.localhost"}, api.Domain{FQDN: "bob.trusted.com"}, api.Domain{FQDN: "gopherpit.localhost"}, api.Domain{FQDN: "localhost", CertificateIgnore: true, Disabled: true}, api.Domain{FQDN: "to-bob.localhost"}},
				"chuck": {api.Domain{FQDN: "chuck.localhost"}},
			}
			for username := range users {
				domains, err := httpClients[username].Domains("", 0)
				if err != nil {
					t.Fatal(err)
				}
				if domains.Count != len(expected[username]) {
					t.Fatalf("%s: expected %q, got %q", username, len(expected[username]), domains.Count)
				}
				if len(domains.Domains) != len(expected[username]) {
					t.Fatalf("%s: expected %q, got %q", username, len(expected[username]), len(domains.Domains))
				}
				for i, d := range domains.Domains {
					e := expected[username][i]
					if d.FQDN != e.FQDN {
						t.Errorf("%s, %d: expected %q, got %q", username, i, e.FQDN, d.FQDN)
					}
					if d.CertificateIgnore != e.CertificateIgnore {
						t.Errorf("%s, %d: expected %v, got %vq", username, i, e.CertificateIgnore, d.CertificateIgnore)
					}
					if d.Disabled != e.Disabled {
						t.Errorf("%s, %d: expected %v, got %v", username, i, e.Disabled, d.Disabled)
					}
					ownerID := users[username].ID
					if d.FQDN == "localhost" || d.FQDN == "gopherpit.localhost" {
						ownerID = users["alice"].ID
					}
					if d.OwnerUserID != ownerID {
						t.Errorf("%s, %d: expected %q, got %q", username, i, ownerID, d.OwnerUserID)
					}
				}
			}
		})
		t.Run("pagination limit 2", func(t *testing.T) {
			page1, err := httpClients["alice"].Domains("", 2)
			if err != nil {
				t.Fatal(err)
			}
			if page1.Count != 2 {
				t.Errorf("expected %v, got %v", 2, page1.Count)
			}
			if len(page1.Domains) != 2 {
				t.Errorf("expected %v, got %v", 2, len(page1.Domains))
			}
			page2, err := httpClients["alice"].Domains(page1.Next, 2)
			if err != nil {
				t.Fatal(err)
			}
			if page2.Count != 2 {
				t.Errorf("expected %v, got %v", 2, page2.Count)
			}
			if len(page2.Domains) != 2 {
				t.Errorf("expected %v, got %v", 2, len(page2.Domains))
			}
			page3, err := httpClients["alice"].Domains(page2.Next, 2)
			if err != nil {
				t.Fatal(err)
			}
			if page3.Count != 2 {
				t.Errorf("expected %v, got %v", 2, page3.Count)
			}
			if len(page3.Domains) != 2 {
				t.Errorf("expected %v, got %v", 2, len(page3.Domains))
			}
			if page3.Previous != page1.Next {
				t.Errorf("expected %q, got %q", page1.Next, page3.Previous)
			}
			page4, err := httpClients["alice"].Domains(page3.Next, 2)
			if err != nil {
				t.Fatal(err)
			}
			if page4.Count != 1 {
				t.Errorf("expected %v, got %v", 1, page4.Count)
			}
			if len(page4.Domains) != 1 {
				t.Errorf("expected %v, got %v", 1, len(page4.Domains))
			}
			if page4.Previous != page2.Next {
				t.Errorf("expected %q, got %q", page2.Next, page4.Previous)
			}
		})
		t.Run("pagination limit 3", func(t *testing.T) {
			page1, err := httpClients["alice"].Domains("", 3)
			if err != nil {
				t.Fatal(err)
			}
			if page1.Count != 3 {
				t.Errorf("expected %v, got %v", 3, page1.Count)
			}
			if len(page1.Domains) != 3 {
				t.Errorf("expected %v, got %v", 3, len(page1.Domains))
			}
			page2, err := httpClients["alice"].Domains(page1.Next, 3)
			if err != nil {
				t.Fatal(err)
			}
			if page2.Count != 3 {
				t.Errorf("expected %v, got %v", 3, page2.Count)
			}
			if len(page2.Domains) != 3 {
				t.Errorf("expected %v, got %v", 3, len(page2.Domains))
			}
			page3, err := httpClients["alice"].Domains(page2.Next, 3)
			if err != nil {
				t.Fatal(err)
			}
			if page3.Count != 1 {
				t.Errorf("expected %v, got %v", 1, page3.Count)
			}
			if len(page3.Domains) != 1 {
				t.Errorf("expected %v, got %v", 1, len(page3.Domains))
			}
			if page3.Previous != page1.Next {
				t.Errorf("expected %q, got %q", page1.Next, page3.Previous)
			}
		})
		t.Run("domain not found", func(t *testing.T) {
			_, err := httpClients["alice"].Domains("unknown.localhost", 0)
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
			_, err = httpClients["chuck"].Domains("localhost", 0)
			if err != api.ErrDomainNotFound {
				t.Errorf("expected %q, got %q", api.ErrDomainNotFound, err)
			}
		})
	})
	t.Run("list domain tokens", func(t *testing.T) {
		tokens, err := httpClients["alice"].DomainTokens("example.com")
		if err != nil {
			t.Fatal(err)
		}
		if len(tokens.Tokens) != 1 {
			t.Errorf("expected %v, got %v", 1, len(tokens.Tokens))
		} else {
			if tokens.Tokens[0].FQDN != "_gopherpit.example.com" {
				t.Errorf("expected %q, got %q", "_gopherpit.example.com", tokens.Tokens[0].FQDN)
			}
			if len(tokens.Tokens[0].Token) != 28 {
				t.Errorf("expected %v, got %v", 28, len(tokens.Tokens[0].Token))
			}
		}
		tokens, err = httpClients["alice"].DomainTokens("subsub.sub.example.com")
		if err != nil {
			t.Fatal(err)
		}
		if len(tokens.Tokens) != 3 {
			t.Errorf("expected %v, got %v", 3, len(tokens.Tokens))
		} else {
			if tokens.Tokens[0].FQDN != "_gopherpit.example.com" {
				t.Errorf("expected %q, got %q", "_gopherpit.example.com", tokens.Tokens[0].FQDN)
			}
			if len(tokens.Tokens[0].Token) != 28 {
				t.Errorf("expected %v, got %v", 28, len(tokens.Tokens[0].Token))
			}
			if tokens.Tokens[1].FQDN != "_gopherpit.sub.example.com" {
				t.Errorf("expected %q, got %q", "_gopherpit.sub.example.com", tokens.Tokens[1].FQDN)
			}
			if len(tokens.Tokens[1].Token) != 28 {
				t.Errorf("expected %v, got %v", 28, len(tokens.Tokens[1].Token))
			}
			if tokens.Tokens[2].FQDN != "_gopherpit.subsub.sub.example.com" {
				t.Errorf("expected %q, got %q", "_gopherpit.subsub.sub.example.com", tokens.Tokens[2].FQDN)
			}
			if len(tokens.Tokens[2].Token) != 28 {
				t.Errorf("expected %v, got %v", 28, len(tokens.Tokens[2].Token))
			}
		}
		t.Run("domain fqdn invalid", func(t *testing.T) {
			for _, fqdn := range []string{
				"localhost",
				"test.invalid-tld",
			} {
				_, err := httpClients["alice"].DomainTokens(fqdn)
				if err != api.ErrDomainFQDNInvalid {
					t.Errorf("%q: expected %q, got %q", fqdn, api.ErrDomainFQDNInvalid, err)
				}
			}
		})
	})
}

func stringSliceEqual(s1, s2 []string) bool {
	for i, e := range s1 {
		if s2[i] != e {
			return false
		}
	}
	return true
}
