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

var templates = map[string][]string{
	"LandingPage":           {"base.html", "cover.html", "login.html", "landing-page.html"},
	"EmailUnvalidated":      {"base.html", "app.html", "email-unvalidated.html"},
	"Dashboard":             {"base.html", "app.html", "changelog-record.html", "dashboard.html"},
	"About":                 {"base.html", "cover.html", "about.html"},
	"AboutPrivate":          {"base.html", "app.html", "about.html"},
	"License":               {"base.html", "cover.html", "license.html"},
	"LicensePrivate":        {"base.html", "app.html", "license.html"},
	"Contact":               {"base.html", "cover.html", "contact.html"},
	"ContactPrivate":        {"base.html", "app.html", "contact.html"},
	"PublicEmailSettings":   {"base.html", "cover.html", "pubilc-email-settings.html"},
	"Login":                 {"base.html", "cover.html", "login.html"},
	"Registration":          {"base.html", "cover.html", "registration.html"},
	"PasswordReset":         {"base.html", "cover.html", "password-reset.html"},
	"PasswordResetToken":    {"base.html", "cover.html", "password-reset-token.html"},
	"EmailValidation":       {"base.html", "app.html", "email-validation.html"},
	"Settings":              {"base.html", "app.html", "settings/settings.html"},
	"SettingsEmail":         {"base.html", "app.html", "settings/email.html"},
	"SettingsNotifications": {"base.html", "app.html", "settings/notifications.html"},
	"SettingsPassword":      {"base.html", "app.html", "settings/password.html"},
	"SettingsDeleteAccount": {"base.html", "app.html", "settings/delete-account.html"},
	// Certificates
	"RegisterACMEUser":        {"base.html", "cover.html", "register-acme-user.html"},
	"RegisterACMEUserPrivate": {"base.html", "app.html", "register-acme-user.html"},
	// Packages
	"PackageResolution": {"package-resolution.html"},
	"DomainPackages":    {"base.html", "app.html", "domain-packages.html"},
	"DomainAdd":         {"base.html", "app.html", "domain-add.html"},
	"DomainSettings":    {"base.html", "app.html", "domain-settings.html"},
	"DomainTeam":        {"base.html", "app.html", "domain-team.html"},
	"DomainUser":        {"base.html", "app.html", "domain-user.html"},
	"DomainChangelog":   {"base.html", "app.html", "changelog-record.html", "domain-changelog.html"},
	"DomainPackageEdit": {"base.html", "app.html", "domain-package-edit.html"},
	"DomainUserGrant":   {"base.html", "app.html", "domain-user-grant.html"},
	"DomainUserRevoke":  {"base.html", "app.html", "domain-user-revoke.html"},
	"DomainOwnerChange": {"base.html", "app.html", "domain-owner-change.html"},
	"UserPage":          {"base.html", "app.html", "user-page.html"},
	// Maintenance
	"Maintenance": {"base.html", "cover.html", "error.html", "error/maintenance.html"},
	// HTTP Errors
	"BadRequest":                   {"base.html", "cover.html", "error.html", "error/bad-request.html"},
	"BadRequestPrivate":            {"base.html", "app.html", "error-private.html", "error/bad-request.html"},
	"Unauthorized":                 {"base.html", "cover.html", "error.html", "error/unauthorized.html"},
	"UnauthorizedPrivate":          {"base.html", "app.html", "error-private.html", "error/unauthorized.html"},
	"Forbidden":                    {"base.html", "cover.html", "error.html", "error/forbidden.html"},
	"ForbiddenPrivate":             {"base.html", "app.html", "error-private.html", "error/forbidden.html"},
	"NotFound":                     {"base.html", "cover.html", "error.html", "error/not-found.html"},
	"NotFoundPrivate":              {"base.html", "app.html", "error-private.html", "error/not-found.html"},
	"RequestEntityTooLarge":        {"base.html", "cover.html", "error.html", "error/request-entity-too-large.html"},
	"RequestEntityTooLargePrivate": {"base.html", "app.html", "error-private.html", "error/request-entity-too-large.html"},
	"InternalServerError":          {"base.html", "cover.html", "error.html", "error/internal-server-error.html"},
	"InternalServerErrorPrivate":   {"base.html", "app.html", "error-private.html", "error/internal-server-error.html"},
	"ServiceUnavailable":           {"base.html", "cover.html", "error.html", "error/service-unavailable.html"},
	"ServiceUnavailablePrivate":    {"base.html", "app.html", "error-private.html", "error/service-unavailable.html"},
}

func (s *Server) template(t string) (tpl *template.Template) {
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
