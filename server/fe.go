// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"
	"sort"

	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	token := ""
	domains := packages.Domains{}
	for {
		response, err := srv.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist || err == packages.DomainNotFound {
				srv.logger.Warningf("dashboard: user domains %s: %s", u.ID, err)
				break
			}
			srv.logger.Errorf("dashboard: user domains %s: %s", u.ID, err)
			htmlInternalServerErrorHandler(w, r)
			return
		}
		domains = append(domains, response.Domains...)
		token = response.Next
		if token == "" {
			break
		}
	}

	if len(domains) == 1 {
		http.Redirect(w, r, "/domain/"+domains[0].FQDN, http.StatusTemporaryRedirect)
		return
	}

	var changelogLimit int
	switch {
	case len(domains) > 5:
		changelogLimit = 2
	case len(domains) > 3:
		changelogLimit = 3
	}

	users := map[string]*user.User{
		u.ID: u,
	}
	cls := changelogs{}
	for _, domain := range domains {
		cl, err := srv.PackagesService.ChangelogForDomain(domain.ID, "", changelogLimit)
		if err != nil {
			if err == packages.DomainNotFound {
				srv.logger.Warningf("dashboard: domain changelog %s: %s", domain.ID, err)
				continue
			}
			srv.logger.Errorf("dashboard: domain changelog %s: %s", domain.ID, err)
			htmlInternalServerErrorHandler(w, r)
			return
		}
		records := make([]changelogRecord, 0, len(cl.Records))
		for _, record := range cl.Records {
			if err = updateChangelogRecords(*u, record, &records, &users); err != nil {
				srv.logger.Errorf("domain: update users map: %s", err)
				htmlInternalServerErrorHandler(w, r)
				return
			}
		}
		cls = append(cls, changelog{
			Records:  records,
			Domain:   domain,
			Previous: cl.Previous,
		})
	}

	sort.Sort(cls)
	respond(w, "Dashboard", map[string]interface{}{
		"User":       u,
		"Domains":    domains,
		"Users":      users,
		"Changelogs": cls,
	})
}

func landingPageHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, "LandingPage", nil)
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		respond(w, "AboutPrivate", map[string]interface{}{
			"User":    u,
			"Version": version(),
		})
		return
	}
	respond(w, "About", map[string]interface{}{
		"Version": version(),
	})
}

func contactHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		u = nil
	}
	if u != nil {
		respond(w, "ContactPrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	respond(w, "Contact", nil)
}

func licenseHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		respond(w, "LicensePrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	respond(w, "License", nil)
}

func docHandler(w http.ResponseWriter, r *http.Request) {
	if !srv.APIEnabled {
		htmlNotFoundHandler(w, r)
		return
	}

	u, r, err := getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		respond(w, "DocPrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	respond(w, "Doc", nil)
}
