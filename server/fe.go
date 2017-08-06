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

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	token := ""
	domains := packages.Domains{}
	for {
		response, err := s.PackagesService.DomainsByUser(u.ID, token, 0)
		if err != nil {
			if err == packages.ErrUserDoesNotExist || err == packages.ErrDomainNotFound {
				s.Logger.Warningf("dashboard: user domains %s: %s", u.ID, err)
				break
			}
			s.Logger.Errorf("dashboard: user domains %s: %s", u.ID, err)
			s.htmlServerError(w, r, err)
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
			if err == packages.ErrDomainNotFound {
				s.Logger.Warningf("dashboard: domain changelog %s: %s", domain.ID, err)
				continue
			}
			s.Logger.Errorf("dashboard: domain changelog %s: %s", domain.ID, err)
			s.htmlServerError(w, r, err)
			return
		}
		records := make([]changelogRecord, 0, len(cl.Records))
		for _, record := range cl.Records {
			if err = s.updateChangelogRecords(*u, record, &records, &users); err != nil {
				s.Logger.Errorf("domain: update users map: %s", err)
				s.htmlServerError(w, r, err)
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
	s.html.Respond(w, "Dashboard", map[string]interface{}{
		"User":       u,
		"Domains":    domains,
		"Users":      users,
		"Changelogs": cls,
	})
}

func (s *Server) landingPageHandler(w http.ResponseWriter, r *http.Request) {
	s.html.Respond(w, "LandingPage", nil)
}

func (s *Server) aboutHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		s.html.Respond(w, "AboutPrivate", map[string]interface{}{
			"User":    u,
			"Version": s.version(),
		})
		return
	}
	s.html.Respond(w, "About", map[string]interface{}{
		"Version": s.version(),
	})
}

func (s *Server) contactHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		u = nil
	}
	if u != nil {
		s.html.Respond(w, "ContactPrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	s.html.Respond(w, "Contact", nil)
}

func (s *Server) licenseHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		s.html.Respond(w, "LicensePrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	s.html.Respond(w, "License", nil)
}

func (s *Server) apiDocsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.APIEnabled {
		s.htmlNotFoundHandler(w, r)
		return
	}

	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	if u != nil {
		s.html.Respond(w, "DocPrivate", map[string]interface{}{
			"User": u,
		})
		return
	}
	s.html.Respond(w, "Doc", nil)
}
