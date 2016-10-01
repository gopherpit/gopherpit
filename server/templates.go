// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"html/template"
	"path/filepath"
	"strings"
)

func (s *Server) templateLandingPage() *template.Template {
	return s.getTemplate("base.html", "cover.html", "login.html", "landing-page.html")
}

func (s *Server) templateEmailUnvalidated() *template.Template {
	return s.getTemplate("base.html", "app.html", "email-unvalidated.html")
}

func (s *Server) templateDashboard() *template.Template {
	return s.getTemplate("base.html", "app.html", "changelog-record.html", "dashboard.html")
}

func (s *Server) templateAbout() *template.Template {
	return s.getTemplate("base.html", "cover.html", "about.html")
}

func (s *Server) templateAboutPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "about.html")
}

func (s *Server) templateLicense() *template.Template {
	return s.getTemplate("base.html", "cover.html", "license.html")
}

func (s *Server) templateLicensePrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "license.html")
}

func (s *Server) templateContact() *template.Template {
	return s.getTemplate("base.html", "cover.html", "contact.html")
}

func (s *Server) templateContactPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "contact.html")
}

func (s *Server) templatePublicEmailSettings() *template.Template {
	return s.getTemplate("base.html", "cover.html", "pubilc-email-settings.html")
}

func (s *Server) templateLogin() *template.Template {
	return s.getTemplate("base.html", "cover.html", "login.html")
}

func (s *Server) templateRegistration() *template.Template {
	return s.getTemplate("base.html", "cover.html", "registration.html")
}

func (s *Server) templatePasswordReset() *template.Template {
	return s.getTemplate("base.html", "cover.html", "password-reset.html")
}

func (s *Server) templatePasswordResetToken() *template.Template {
	return s.getTemplate("base.html", "cover.html", "password-reset-token.html")
}

func (s *Server) templateEmailValidation() *template.Template {
	return s.getTemplate("base.html", "app.html", "email-validation.html")
}

func (s *Server) templateSettings() *template.Template {
	return s.getTemplate("base.html", "app.html", "settings/settings.html")
}

func (s *Server) templateSettingsEmail() *template.Template {
	return s.getTemplate("base.html", "app.html", "settings/email.html")
}

func (s *Server) templateSettingsNotifications() *template.Template {
	return s.getTemplate("base.html", "app.html", "settings/notifications.html")
}

func (s *Server) templateSettingsPassword() *template.Template {
	return s.getTemplate("base.html", "app.html", "settings/password.html")
}

func (s *Server) templateSettingsDeleteAccount() *template.Template {
	return s.getTemplate("base.html", "app.html", "settings/delete-account.html")
}

// Certificates

func (s *Server) templateRegisterACMEUser() *template.Template {
	return s.getTemplate("base.html", "cover.html", "register-acme-user.html")
}

func (s *Server) templateRegisterACMEUserPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "register-acme-user.html")
}

// Packages

func (s *Server) templatePackageResolution() *template.Template {
	return s.getTemplate("package-resolution.html")
}

func (s *Server) templateDomainPackages() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-packages.html")
}

func (s *Server) templateDomainAdd() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-add.html")
}

func (s *Server) templateDomainSettings() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-settings.html")
}

func (s *Server) templateDomainTeam() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-team.html")
}

func (s *Server) templateDomainUser() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-user.html")
}

func (s *Server) templateDomainChangelog() *template.Template {
	return s.getTemplate("base.html", "app.html", "changelog-record.html", "domain-changelog.html")
}

func (s *Server) templateDomainPackageEdit() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-package-edit.html")
}

func (s *Server) templateDomainUserGrant() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-user-grant.html")
}

func (s *Server) templateDomainUserRevoke() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-user-revoke.html")
}

func (s *Server) templateDomainOwnerChange() *template.Template {
	return s.getTemplate("base.html", "app.html", "domain-owner-change.html")
}

func (s *Server) templateUserPage() *template.Template {
	return s.getTemplate("base.html", "app.html", "user-page.html")
}

// Maintenance

func (s *Server) templateMaintenance() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/maintenance.html")
}

// HTTP Errors

func (s *Server) templateBadRequest() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/bad-request.html")
}

func (s *Server) templateBadRequestPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/bad-request.html")
}

func (s *Server) templateUnauthorized() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/unauthorized.html")
}

func (s *Server) templateUnauthorizedPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/unauthorized.html")
}

func (s *Server) templateForbidden() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/forbidden.html")
}

func (s *Server) templateForbiddenPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/forbidden.html")
}

func (s *Server) templateNotFound() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/not-found.html")
}

func (s *Server) templateNotFoundPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/not-found.html")
}

func (s *Server) templateRequestEntityTooLarge() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/request-entity-too-large.html")
}

func (s *Server) templateRequestEntityTooLargePrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/request-entity-too-large.html")
}

func (s *Server) templateInternalServerError() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/internal-server-error.html")
}

func (s *Server) templateInternalServerErrorPrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/internal-server-error.html")
}

func (s *Server) templateServiceUnavailable() *template.Template {
	return s.getTemplate("base.html", "cover.html", "error.html", "error/service-unavaliable.html")
}

func (s *Server) templateServiceUnavailablePrivate() *template.Template {
	return s.getTemplate("base.html", "app.html", "error-private.html", "error/service-unavaliable.html")
}

func (s Server) newTemplate(files ...string) (*template.Template, error) {
	fs := []string{}
	for _, f := range files {
		fs = append(fs, filepath.Join(s.TemplatesDir, f))
	}
	return template.New("*").Funcs(s.templateFunctions).Delims("[[", "]]").ParseFiles(fs...)
}

func (s *Server) getTemplate(files ...string) (t *template.Template) {
	key := strings.Join(files, "\n")
	var ok bool
	t, ok = s.templateCache[key]
	if ok {
		return
	}
	var err error
	t, err = s.newTemplate(files...)
	if err != nil {
		panic(err)
	}
	s.mu.Lock()
	s.templateCache[key] = t
	s.mu.Unlock()
	return
}
