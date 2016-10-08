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

type tid int // Template ID

const (
	tidLandingPage tid = iota
	tidEmailUnvalidated
	tidDashboard
	tidAbout
	tidAboutPrivate
	tidLicense
	tidLicensePrivate
	tidContact
	tidContactPrivate
	tidPublicEmailSettings
	tidLogin
	tidRegistration
	tidPasswordReset
	tidPasswordResetToken
	tidEmailValidation
	tidSettings
	tidSettingsEmail
	tidSettingsNotifications
	tidSettingsPassword
	tidSettingsDeleteAccount
	// Certificates
	tidRegisterACMEUser
	tidRegisterACMEUserPrivate
	// Packages
	tidPackageResolution
	tidDomainPackages
	tidDomainAdd
	tidDomainSettings
	tidDomainTeam
	tidDomainUser
	tidDomainChangelog
	tidDomainPackageEdit
	tidDomainUserGrant
	tidDomainUserRevoke
	tidDomainOwnerChange
	tidUserPage
	// Maintenance
	tidMaintenance
	// HTTP Errors
	tidBadRequest
	tidBadRequestPrivate
	tidUnauthorized
	tidUnauthorizedPrivate
	tidForbidden
	tidForbiddenPrivate
	tidNotFound
	tidNotFoundPrivate
	tidRequestEntityTooLarge
	tidRequestEntityTooLargePrivate
	tidInternalServerError
	tidInternalServerErrorPrivate
	tidServiceUnavailable
	tidServiceUnavailablePrivate
)

var (
	templates = map[tid][]string{
		tidLandingPage:           {"base.html", "cover.html", "login.html", "landing-page.html"},
		tidEmailUnvalidated:      {"base.html", "app.html", "email-unvalidated.html"},
		tidDashboard:             {"base.html", "app.html", "changelog-record.html", "dashboard.html"},
		tidAbout:                 {"base.html", "cover.html", "about.html"},
		tidAboutPrivate:          {"base.html", "app.html", "about.html"},
		tidLicense:               {"base.html", "cover.html", "license.html"},
		tidLicensePrivate:        {"base.html", "app.html", "license.html"},
		tidContact:               {"base.html", "cover.html", "contact.html"},
		tidContactPrivate:        {"base.html", "app.html", "contact.html"},
		tidPublicEmailSettings:   {"base.html", "cover.html", "pubilc-email-settings.html"},
		tidLogin:                 {"base.html", "cover.html", "login.html"},
		tidRegistration:          {"base.html", "cover.html", "registration.html"},
		tidPasswordReset:         {"base.html", "cover.html", "password-reset.html"},
		tidPasswordResetToken:    {"base.html", "cover.html", "password-reset-token.html"},
		tidEmailValidation:       {"base.html", "app.html", "email-validation.html"},
		tidSettings:              {"base.html", "app.html", "settings/settings.html"},
		tidSettingsEmail:         {"base.html", "app.html", "settings/email.html"},
		tidSettingsNotifications: {"base.html", "app.html", "settings/notifications.html"},
		tidSettingsPassword:      {"base.html", "app.html", "settings/password.html"},
		tidSettingsDeleteAccount: {"base.html", "app.html", "settings/delete-account.html"},
		// Certificates
		tidRegisterACMEUser:        {"base.html", "cover.html", "register-acme-user.html"},
		tidRegisterACMEUserPrivate: {"base.html", "app.html", "register-acme-user.html"},
		// Packages
		tidPackageResolution: {"package-resolution.html"},
		tidDomainPackages:    {"base.html", "app.html", "domain-packages.html"},
		tidDomainAdd:         {"base.html", "app.html", "domain-add.html"},
		tidDomainSettings:    {"base.html", "app.html", "domain-settings.html"},
		tidDomainTeam:        {"base.html", "app.html", "domain-team.html"},
		tidDomainUser:        {"base.html", "app.html", "domain-user.html"},
		tidDomainChangelog:   {"base.html", "app.html", "changelog-record.html", "domain-changelog.html"},
		tidDomainPackageEdit: {"base.html", "app.html", "domain-package-edit.html"},
		tidDomainUserGrant:   {"base.html", "app.html", "domain-user-grant.html"},
		tidDomainUserRevoke:  {"base.html", "app.html", "domain-user-revoke.html"},
		tidDomainOwnerChange: {"base.html", "app.html", "domain-owner-change.html"},
		tidUserPage:          {"base.html", "app.html", "user-page.html"},
		// Maintenance
		tidMaintenance: {"base.html", "cover.html", "error.html", "error/maintenance.html"},
		// HTTP Errors
		tidBadRequest:                   {"base.html", "cover.html", "error.html", "error/bad-request.html"},
		tidBadRequestPrivate:            {"base.html", "app.html", "error-private.html", "error/bad-request.html"},
		tidUnauthorized:                 {"base.html", "cover.html", "error.html", "error/unauthorized.html"},
		tidUnauthorizedPrivate:          {"base.html", "app.html", "error-private.html", "error/unauthorized.html"},
		tidForbidden:                    {"base.html", "cover.html", "error.html", "error/forbidden.html"},
		tidForbiddenPrivate:             {"base.html", "app.html", "error-private.html", "error/forbidden.html"},
		tidNotFound:                     {"base.html", "cover.html", "error.html", "error/not-found.html"},
		tidNotFoundPrivate:              {"base.html", "app.html", "error-private.html", "error/not-found.html"},
		tidRequestEntityTooLarge:        {"base.html", "cover.html", "error.html", "error/request-entity-too-large.html"},
		tidRequestEntityTooLargePrivate: {"base.html", "app.html", "error-private.html", "error/request-entity-too-large.html"},
		tidInternalServerError:          {"base.html", "cover.html", "error.html", "error/internal-server-error.html"},
		tidInternalServerErrorPrivate:   {"base.html", "app.html", "error-private.html", "error/internal-server-error.html"},
		tidServiceUnavailable:           {"base.html", "cover.html", "error.html", "error/service-unavailable.html"},
		tidServiceUnavailablePrivate:    {"base.html", "app.html", "error-private.html", "error/service-unavailable.html"},
	}
)

func (s *Server) template(t tid) (tpl *template.Template) {
	key := strings.Join(templates[t], "\n")
	var ok bool
	s.mu.RLock()
	tpl, ok = s.templateCache[key]
	s.mu.RUnlock()
	if ok {
		return
	}
	var err error

	fs := []string{}
	for _, f := range templates[t] {
		fs = append(fs, filepath.Join(s.TemplatesDir, f))
	}
	tpl, err = template.New("*").Funcs(s.templateFunctions).Delims("[[", "]]").ParseFiles(fs...)

	if err != nil {
		panic(err)
	}
	s.mu.Lock()
	s.templateCache[key] = tpl
	s.mu.Unlock()
	return
}
