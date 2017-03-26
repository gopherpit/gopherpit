// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/base32"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
)

var templates = map[string][]string{
	"LandingPage":           {"base.html", "cover.html", "login.html", "landing-page.html"},
	"EmailUnvalidated":      {"base.html", "app.html", "email-unvalidated.html"},
	"Dashboard":             {"base.html", "app.html", "changelog-record.html", "dashboard.html"},
	"About":                 {"base.html", "cover.html", "about.html"},
	"AboutPrivate":          {"base.html", "app.html", "about.html"},
	"License":               {"base.html", "cover.html", "license.html"},
	"LicensePrivate":        {"base.html", "app.html", "license.html"},
	"Doc":                   {"base.html", "public.html", "docs-api.html"},
	"DocPrivate":            {"base.html", "app.html", "docs-api.html"},
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
	"SettingsAPIAccess":     {"base.html", "app.html", "settings/api-access.html"},
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

func assetFunc(str string) string {
	p, err := srv.assetsServer.HashedPath(str)
	if err != nil {
		srv.Logger.Errorf("html response: asset func: hashed path: %s", err)
		return str
	}
	return p
}

func relativeTimeFunc(t time.Time) string {
	const day = 24 * time.Hour
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < 2*time.Second:
		return "one second ago"
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", d/time.Second)
	case d < 2*time.Minute:
		return "one minute ago"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", d/time.Minute)
	case d < 2*time.Hour:
		return "one hour ago"
	case d < day:
		return fmt.Sprintf("%d hours ago", d/time.Hour)
	case d < 2*day:
		return "one day ago"
	}
	return fmt.Sprintf("%d days ago", d/day)
}

func safeHTMLFunc(text string) template.HTML {
	return template.HTML(text)
}

func yearRangeFunc(year int) string {
	curYear := time.Now().Year()
	if year >= curYear {
		return fmt.Sprintf("%d", year)
	}
	return fmt.Sprintf("%d - %d", year, curYear)
}

func containsStringFunc(list []string, element, yes, no string) string {
	for _, e := range list {
		if e == element {
			return yes
		}
	}
	return no
}

func htmlBrFunc(text string) string {
	text = template.HTMLEscapeString(text)
	return strings.Replace(text, "\n", "<br>", -1)
}

func mapFunc(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("map keys must be strings")
		}
		m[key] = values[i+1]
	}
	return m, nil
}

func newContext(m map[string]interface{}) func(string) interface{} {
	return func(key string) interface{} {
		if value, ok := m[key]; ok {
			return value
		}
		return nil
	}
}

func base32encodeFunc(text string) string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString([]byte(text)), "=")
}
