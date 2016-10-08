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

func (s Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	token := ""
	domains := packages.Domains{}
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.UserDoesNotExist || err == packages.DomainNotFound {
				s.logger.Warningf("dashboard: user domains %s: %s", u.ID, err)
				break
			}
			s.logger.Errorf("dashboard: user domains %s: %s", u.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
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
		cl, err := s.PackagesService.ChangelogForDomain(domain.ID, "", changelogLimit)
		if err != nil {
			if err == packages.DomainNotFound {
				s.logger.Warningf("dashboard: domain changelog %s: %s", domain.ID, err)
				continue
			}
			s.logger.Errorf("dashboard: domain changelog %s: %s", domain.ID, err)
			s.htmlInternalServerErrorHandler(w, r)
			return
		}
		records := make([]changelogRecord, 0, len(cl.Records))
		for _, record := range cl.Records {
			if err = s.updateChangelogRecords(*u, record, &records, &users); err != nil {
				s.logger.Errorf("domain: update users map: %s", err)
				s.htmlInternalServerErrorHandler(w, r)
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
	respond(w, s.template(tidDashboard), map[string]interface{}{
		"User":       u,
		"Domains":    domains,
		"Users":      users,
		"Changelogs": cls,
	})
}

func (s Server) landingPageHandler(w http.ResponseWriter, r *http.Request) {
	respond(w, s.template(tidLandingPage), nil)
}

func (s Server) aboutHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		respond(w, s.template(tidAboutPrivate), map[string]interface{}{
			"User":    u,
			"Version": s.Version(),
		})
		return
	}
	respond(w, s.template(tidAbout), map[string]interface{}{
		"Version": s.Version(),
	})
}

func (s Server) contactHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		u = nil
	}
	if u != nil {
		respond(w, s.template(tidContactPrivate), map[string]interface{}{
			"User": u,
		})
		return
	}
	respond(w, s.template(tidContact), nil)
}

func (s Server) licenseHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.user(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		respond(w, s.template(tidLicensePrivate), map[string]interface{}{
			"User": u,
		})
		return
	}
	respond(w, s.template(tidLicense), nil)
}
