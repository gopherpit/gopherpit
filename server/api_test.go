// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"gopherpit.com/gopherpit/api"
	"gopherpit.com/gopherpit/client"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/user"
)

func TestAPIAccess(t *testing.T) {
	if err := startTestServer(nil); err != nil {
		t.Fatal(err)
	}
	defer stopTestServer()

	t.Run("invalid key", func(t *testing.T) {
		c := client.NewClientWithEndpoint(
			"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
			"INVALIDKEY",
		)
		c.UserAgent = "gopherpit-test-client"

		_, err := c.Domains("", 0)
		if err != api.ErrUnauthorized {
			t.Errorf("expected %q, got %q", api.ErrUnauthorized, err)
		}

		c = client.NewClientWithEndpoint(
			"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
			"",
		)
		c.UserAgent = "gopherpit-test-client"

		_, err = c.Domains("", 0)
		if err != api.ErrUnauthorized {
			t.Errorf("expected %q, got %q", api.ErrUnauthorized, err)
		}
	})

	t.Run("wrong ip", func(t *testing.T) {
		username := "alice"
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

		_, ipV4Net, err := net.ParseCIDR("127.0.0.2/32")
		if err != nil {
			t.Fatalf("parse IPv4 net: %s", err)
		}

		k, err := srv.KeyService.CreateKey(u.ID, &key.Options{
			AuthorizedNetworks: &[]net.IPNet{
				*ipV4Net,
			},
		})
		if err != nil {
			t.Fatalf("create key: %s", err)
		}

		c := client.NewClientWithEndpoint(
			"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
			k.Secret,
		)
		c.UserAgent = username + "-gopherpit-test-client"

		_, err = c.Domains("", 0)
		if err != api.ErrUnauthorized {
			t.Errorf("expected %q, got %q", api.ErrUnauthorized, err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		username := "bob"
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

		c := client.NewClientWithEndpoint(
			"localhost:"+strconv.Itoa(srv.port)+"/api/",
			k.Secret,
		)
		c.UserAgent = username + "-gopherpit-test-client"

		_, err = c.Domains("", 0)
		if err != api.ErrNotFound {
			t.Errorf("expected %q, got %q", api.ErrNotFound, err)
		}

		c = client.NewClientWithEndpoint(
			"localhost:"+strconv.Itoa(srv.port)+"/api/v2/",
			k.Secret,
		)
		c.UserAgent = username + "-gopherpit-test-client"

		_, err = c.Domains("", 0)
		if err != api.ErrNotFound {
			t.Errorf("expected %q, got %q", api.ErrNotFound, err)
		}
	})
}

func TestAPIRateLimit(t *testing.T) {
	if err := startTestServer(map[string]interface{}{
		"APIHourlyRateLimit": 5,
	}); err != nil {
		t.Fatal(err)
	}
	defer stopTestServer()

	username := "alice"
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

	c := client.NewClientWithEndpoint(
		"localhost:"+strconv.Itoa(srv.port)+"/api/v1",
		k.Secret,
	)
	c.UserAgent = username + "-gopherpit-test-client"

	for _, fqdn := range []string{
		"1.localhost",
		"2.localhost",
		"3.localhost",
		"4.localhost",
		"5.localhost",
	} {
		domain, err := c.AddDomain(&api.DomainOptions{
			FQDN: &fqdn,
		})
		if err != nil {
			t.Fatal(err)
		}
		if domain.FQDN != fqdn {
			t.Errorf("expected %q, got %q", fqdn, domain.FQDN)
		}
	}
	fqdn := "6.localhost"
	_, err = c.AddDomain(&api.DomainOptions{
		FQDN: &fqdn,
	})
	if err != api.ErrTooManyRequests {
		t.Errorf("expected %q, got %q", api.ErrTooManyRequests, err)
	}
}
