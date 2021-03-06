// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/throttled/throttled.v2/store/memstore"
	"resenje.org/logging"
	"resenje.org/recovery"
	"resenje.org/web/maintenance"

	"gopherpit.com/gopherpit/server/config"
	"gopherpit.com/gopherpit/services/key/bolt"
	"gopherpit.com/gopherpit/services/packages/bolt"
	"gopherpit.com/gopherpit/services/session/bolt"
	"gopherpit.com/gopherpit/services/user/bolt"
)

// testServerOptions encapsulates all options needed for server and services
// for server to run for testing. It provides a convenient methods to
// update it's fields.
type testServerOptions struct {
	Options
}

func (o *testServerOptions) set(field string, value interface{}) (isSet bool) {
	f := reflect.ValueOf(o).Elem().FieldByName(field)
	if !f.IsValid() {
		return
	}
	if !f.CanSet() {
		return
	}
	v := reflect.ValueOf(value)
	if f.Kind() != v.Kind() {
		return
	}
	f.Set(v)
	return true
}

func (o *testServerOptions) update(values map[string]interface{}) (notSet []string) {
	for field, value := range values {
		if isSet := o.set(field, value); !isSet {
			notSet = append(notSet, field)
		}
	}
	return
}

// Email recorder is a simple resenje.org/email.Service implementation that
// records values from the latest sent email message.
type emailRecorder struct {
	DefaultFrom     string
	NotifyAddresses []string
	SubjectPrefix   string

	From    string
	To      []string
	Subject string
	Body    string
}

func (r emailRecorder) SendEmail(from string, to []string, subject string, body string) error {
	r.From = from
	r.To = to
	r.Subject = subject
	r.Body = body
	return nil
}

func (r emailRecorder) Notify(subject, body string) error {
	if len(r.NotifyAddresses) == 0 {
		r.From = ""
		r.To = nil
		r.Subject = ""
		r.Body = ""
		return nil
	}
	return r.SendEmail(r.DefaultFrom, r.NotifyAddresses, r.SubjectPrefix+subject, body)
}

// newTestServer requires stopping it:
//
// Example:
//
//     newTestServer(nil)
//     defer s.stopTestServer()
//
func newTestServer(options map[string]interface{}) (*Server, error) {
	logger, err := logging.GetLogger("default")
	if err != nil {
		return nil, fmt.Errorf("get default logger: %s", err)
	}

	storageDir, err := ioutil.TempDir("", "gopherpit-test-")
	if err != nil {
		return nil, fmt.Errorf("temp storage dir: %s", err)
	}

	emailService := &emailRecorder{
		DefaultFrom:     "gopherpit-test@localhost",
		NotifyAddresses: []string{"gopherpit@localhost"},
		SubjectPrefix:   "GopherPitTest",
	}
	recoveryService := &recovery.Service{
		Version:   config.Version,
		BuildInfo: config.BuildInfo,
		LogFunc:   logging.Error,
		Notifier:  emailService,
	}

	sessionDB, err := boltSession.NewDB(filepath.Join(storageDir, "session.db"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("session service bolt database: %s", err)
	}
	sessionService := &boltSession.Service{
		DB:              sessionDB,
		DefaultLifetime: 45 * 24 * time.Hour,
		CleanupPeriod:   0,
		Logger:          logger,
	}
	userDB, err := boltUser.NewDB(filepath.Join(storageDir, "user.db"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("user service bolt database: %s", err)
	}
	userService := &boltUser.Service{
		DB: userDB,
		PasswordNoReuseMonths: 0,
		Logger:                logger,
	}
	packagesDB, err := boltPackages.NewDB(filepath.Join(storageDir, "packages.db"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("packages service bolt database: %s", err)
	}
	packagesChangelog, err := boltPackages.NewChangelogPool(filepath.Join(storageDir, "changelog"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("packages service bolt changelog database pool: %s", err)
	}
	packagesService := &boltPackages.Service{
		DB:        packagesDB,
		Changelog: packagesChangelog,
		Logger:    logger,
	}
	keyDB, err := boltKey.NewDB(filepath.Join(storageDir, "key.db"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("key service bolt database: %s", err)
	}
	keyService := &boltKey.Service{
		DB:     keyDB,
		Logger: logger,
	}
	gcraStoreService, err := memstore.New(65536)
	if err != nil {
		return nil, fmt.Errorf("gcra memstore: %s", err)
	}

	o := testServerOptions{
		Options: Options{
			Name:                    config.Name,
			Version:                 "0",
			BuildInfo:               "test",
			Brand:                   "GopherPit",
			Listen:                  "127.0.0.1:",
			ListenTLS:               "",
			ListenInternal:          "",
			ListenInternalTLS:       "",
			TLSKey:                  "",
			TLSCert:                 "",
			Domain:                  "localhost",
			Headers:                 map[string]string{},
			SessionCookieName:       "testsesid",
			StorageDir:              "",
			GoogleAnalyticsID:       "",
			RememberMeDays:          45,
			DefaultFrom:             "gopherpit-test@localhost",
			ContactRecipientEmail:   "gopherpit-test@localhost",
			ACMEDirectoryURL:        "",
			ACMEDirectoryURLStaging: "",
			SkipDomainVerification:  false,
			VerificationSubdomain:   "_gopherpit",
			TrustedDomains: []string{
				"trusted.com",
			},
			ForbiddenDomains: []string{
				"forbidden.com",
			},
			APITrustedProxyCIDRs: []string{},
			APIProxyRealIPHeader: "X-Real-Ip",
			APIHourlyRateLimit:   0,
			APIEnabled:           true,

			Logger:              logger,
			AccessLogger:        logger,
			AuditLogger:         logger,
			PackageAccessLogger: logger,

			EmailService:       emailService,
			RecoveryService:    recoveryService,
			MaintenanceService: maintenance.New(),

			SessionService:      sessionService,
			UserService:         userService,
			NotificationService: nil,
			CertificateService:  nil,
			PackagesService:     packagesService,
			KeyService:          keyService,
			GCRAStoreService:    gcraStoreService,
		},
	}

	if notSet := o.update(options); notSet != nil {
		return nil, fmt.Errorf("options not set: %s", strings.Join(notSet, ", "))
	}

	// StorageDir must be set explicitly by overriding any configured
	// path as it is removed in s.stopTestServer().
	o.StorageDir = storageDir

	s, err := New(o.Options)
	if err != nil {
		return nil, fmt.Errorf("configure: %s", err)
	}

	if err := s.Serve(); err != nil {
		return nil, fmt.Errorf("serve %s", err)
	}

	return s, nil
}

// s.stopTestServer must be called to shut down HTTP servers and remove storage dir.
func (s *Server) stopTestServer() {
	storageDir := s.StorageDir
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.Shutdown(ctx)
	cancel()
	if storageDir != "" {
		if err := os.RemoveAll(storageDir); err != nil {
			panic(fmt.Errorf("remove %s: %s", storageDir, err))
		}
	}
}

func TestVersionFunc(t *testing.T) {
	t.Run("no version", func(t *testing.T) {
		Version := "0"
		s, err := newTestServer(map[string]interface{}{
			"Version":   "",
			"BuildInfo": "",
		})
		if err != nil {
			t.Fatal(err)
		}
		defer s.stopTestServer()

		v := s.version()
		if v != Version {
			t.Errorf("expected %q, got %q", Version, v)
		}
	})
	t.Run("without build info", func(t *testing.T) {
		Version := "1.25.84"
		s, err := newTestServer(map[string]interface{}{
			"Version":   Version,
			"BuildInfo": "",
		})
		if err != nil {
			t.Fatal(err)
		}
		defer s.stopTestServer()

		v := s.version()
		if v != Version {
			t.Errorf("expected %q, got %q", Version, v)
		}
	})
	t.Run("with build info", func(t *testing.T) {
		Version := "1.25.84-123456"
		s, err := newTestServer(map[string]interface{}{
			"Version":   "1.25.84",
			"BuildInfo": "123456",
		})
		if err != nil {
			t.Fatal(err)
		}
		defer s.stopTestServer()

		v := s.version()
		if v != Version {
			t.Errorf("expected %q, got %q", Version, v)
		}
	})
}
