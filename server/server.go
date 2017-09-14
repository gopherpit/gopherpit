// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server // import "gopherpit.com/gopherpit/server"

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	throttled "gopkg.in/throttled/throttled.v2"
	"resenje.org/logging"
	"resenje.org/recovery"
	"resenje.org/web/file-server"
	"resenje.org/web/maintenance"
	"resenje.org/web/servers"
	"resenje.org/web/servers/http"
	"resenje.org/web/templates"

	"gopherpit.com/gopherpit/pkg/certificate-cache"
	"gopherpit.com/gopherpit/server/data/assets"
	dataTemplates "gopherpit.com/gopherpit/server/data/templates"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/gcrastore"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/notification"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/session"
	"gopherpit.com/gopherpit/services/user"
)

type options = Options

// Server contains all required properties, services and functions
// to provide core functionality.
type Server struct {
	options
	servers *servers.Servers

	startTime time.Time

	certificateCache certificateCache.Cache

	salt []byte

	tlsEnabled       bool
	registerACMEUser bool

	html *templates.Templates

	apiRateLimiter *throttled.GCRARateLimiter
}

// EmailService defines interface for sending email messages.
type EmailService interface {
	Notify(title, body string) error
	SendEmail(from string, to []string, subject string, body string) error
}

// Options structure contains server's configurable properties.
type Options struct {
	Name                    string
	Version                 string
	BuildInfo               string
	Brand                   string
	Listen                  string
	ListenTLS               string
	ListenInternal          string
	ListenInternalTLS       string
	TLSKey                  string
	TLSCert                 string
	Domain                  string
	Headers                 map[string]string
	SessionCookieName       string
	StorageDir              string
	GoogleAnalyticsID       string
	RememberMeDays          int
	DefaultFrom             string
	ContactRecipientEmail   string
	ACMEDirectoryURL        string
	ACMEDirectoryURLStaging string
	SkipDomainVerification  bool
	VerificationSubdomain   string
	TrustedDomains          []string
	ForbiddenDomains        []string
	APITrustedProxyCIDRs    []string
	APIProxyRealIPHeader    string
	APIHourlyRateLimit      int
	APIEnabled              bool

	Logger              *logging.Logger
	AccessLogger        *logging.Logger
	AuditLogger         *logging.Logger
	PackageAccessLogger *logging.Logger

	EmailService       EmailService
	RecoveryService    *recovery.Service
	MaintenanceService *maintenance.Service

	SessionService      session.Service
	UserService         user.Service
	NotificationService notification.Service
	CertificateService  certificate.Service
	PackagesService     packages.Service
	KeyService          key.Service
	GCRAStoreService    gcrastore.Service
}

func (o *Options) version() string {
	if o.BuildInfo != "" {
		return fmt.Sprintf("%s-%s", o.Version, o.BuildInfo)
	}
	return o.Version
}

// New initializes new server with provided options.
func New(o Options) (s *Server, err error) {
	if o.Name == "" {
		o.Name = "server"
	}
	if o.Version == "" {
		o.Version = "0"
	}
	if o.VerificationSubdomain == "" {
		o.VerificationSubdomain = "_" + o.Name
	}
	s = &Server{
		options:          o,
		certificateCache: certificateCache.NewCache(o.CertificateService, 15*time.Minute, time.Minute),
		startTime:        time.Now(),
		tlsEnabled:       o.ListenTLS != "",
		registerACMEUser: o.ListenTLS != "",
		servers: servers.New(
			servers.WithLogger(o.Logger),
			servers.WithRecoverFunc(o.RecoveryService.Recover),
		),
	}

	// Load or generate a salt value.
	saltFilename := filepath.Join(s.StorageDir, s.Name+".salt")
	s.salt, err = ioutil.ReadFile(saltFilename)
	if err != nil && !os.IsNotExist(err) {
		err = fmt.Errorf("read salt file %s: %s", saltFilename, err)
		return
	}
	if len(s.salt) == 0 {
		salt := make([]byte, 48)
		_, err = rand.Read(salt)
		if err != nil {
			err = fmt.Errorf("generate new salt: %s", err)
			return
		}
		s.Logger.Infof("saving new salt to file %s", saltFilename)
		if err = ioutil.WriteFile(saltFilename, salt, 0600); err != nil {
			err = fmt.Errorf("saving salt %s: %s", saltFilename, err)
			return
		}
		s.salt = salt
	}

	// Create assets server
	assetsServer := fileServer.New("/assets", "", &fileServer.Options{
		Hasher:                fileServer.MD5Hasher{HashLength: 8},
		NoHashQueryStrings:    true,
		RedirectTrailingSlash: true,
		IndexPage:             "index.html",
		Filesystem: &assetfs.AssetFS{
			Asset:     assets.Asset,
			AssetDir:  assets.AssetDir,
			AssetInfo: assets.AssetInfo,
		},
		NotFoundHandler:            http.HandlerFunc(s.htmlNotFoundHandler),
		ForbiddenHandler:           http.HandlerFunc(s.htmlForbiddenHandler),
		InternalServerErrorHandler: http.HandlerFunc(s.htmlInternalServerErrorHandler),
	})

	// Parse static HTML documents used as loadable fragments in templates
	fragments, err := parseMarkdownData("fragments")
	if err != nil {
		return nil, fmt.Errorf("parse fragments: %v", err)
	}

	s.html, err = templates.New(
		templates.WithContentType("text/html; charset=utf-8"),
		templates.WithDelims("[[", "]]"),
		templates.WithLogFunc(s.Logger.Errorf),
		templates.WithFunction("asset", func(str string) string {
			p, err := assetsServer.HashedPath(str)
			if err != nil {
				s.Logger.Errorf("html response: asset func: hashed path: %s", err)
				return str
			}
			return p
		}),
		templates.WithFunction("context", templates.NewContextFunc(map[string]interface{}{
			"GoogleAnalyticsID": o.GoogleAnalyticsID,
			"AliasCNAME":        "alias." + o.Domain,
		})),
		templates.WithFunction("fragment", templates.NewContextFunc(fragments)),
		templates.WithFunction("base32encode", func(text string) string {
			return strings.TrimRight(base32.StdEncoding.EncodeToString([]byte(text)), "=")
		}),
		templates.WithFunction("is_gopherpit_domain", func(domain string) bool {
			return strings.HasSuffix(domain, "."+o.Domain)
		}),
		templates.WithFileReadFunc(dataTemplates.Asset),
		templates.WithTemplatesFromFiles(htmlTemplates),
	)
	if err != nil {
		return nil, fmt.Errorf("templates: %v", err)
	}

	s.MaintenanceService.JSON.Body = `{"message":"maintenance","code":503}`
	s.MaintenanceService.Text.Body = "Maintenance"
	s.MaintenanceService.HTML.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, err := s.html.Render("Maintenance", nil)
		if err != nil {
			s.Logger.Errorf("html maintenance render: %s", err)
			m = "Maintenance"
		}
		w.Header().Set("Content-Type", maintenance.HTMLContentType)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, m)
	})

	// API rate limiter
	if s.APIHourlyRateLimit > 0 {
		s.apiRateLimiter, err = throttled.NewGCRARateLimiter(
			s.GCRAStoreService,
			throttled.RateQuota{
				MaxRate:  throttled.PerHour(1),
				MaxBurst: s.APIHourlyRateLimit - 1,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("api rate limiter: %s", err)
		}
	}

	tlsConfig, err := newTLSConfig(s)
	if err != nil {
		return nil, fmt.Errorf("TLS config: %v", err)
	}

	handler := newRouter(s, assetsServer)
	internalHandler := newInternalRouter(s)

	if o.Listen != "" {
		h, err := s.redirectHandler(handler)
		if err != nil {
			return nil, err
		}
		s.servers.Add("HTTP", o.Listen, httpServer.New(
			s.nilRecoveryHandler(s.packageHandler(h)),
		))
	}
	if o.ListenTLS != "" {
		s.servers.Add("TLS HTTP", o.ListenTLS, httpServer.New(
			s.nilRecoveryHandler(s.packageHandler(handler)),
			httpServer.WithTLSConfig(tlsConfig),
		))
	}
	if o.ListenInternal != "" {
		s.servers.Add("internal HTTP", o.ListenInternal, httpServer.New(
			s.nilRecoveryHandler(internalHandler),
		))
	}
	if o.ListenInternalTLS != "" {
		s.servers.Add("internal TLS HTTP", o.ListenInternalTLS, httpServer.New(
			s.nilRecoveryHandler(internalHandler),
			httpServer.WithTLSConfig(tlsConfig),
		))
	}

	return
}

func newTLSConfig(s *Server) (tlsConfig *tls.Config, err error) {
	tlsConfig = &tls.Config{
		MinVersion:         tls.VersionTLS10,
		NextProtos:         []string{"h2"},
		ClientSessionCache: tls.NewLRUClientSessionCache(-1),
	}
	tlsConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// If ServerName is defined in Options as Domain and there is TLSCert in Options
		// use static configuration by returning nil or both cert and err
		if clientHello.ServerName == s.Domain && s.TLSCert != "" {
			return nil, nil
		}
		// Get certificate for this ServerName
		c, err := s.certificateCache.Certificate(clientHello.ServerName)
		switch err {
		case certificateCache.ErrCertificateNotFound, certificate.ErrCertificateNotFound:
			// If ServerName is the same as configured domain or it's www subdomain
			// and tls listener is on https port 443, try to obtain the certificate.
			if strings.HasSuffix(s.ListenTLS, ":443") && (clientHello.ServerName == s.Domain || clientHello.ServerName == "www."+s.Domain) {
				obtainCertificate := false
				// Check if there is not already a request for new certificate active.
				for i := 0; i < 50; i++ {
					yes, err := s.CertificateService.IsCertificateBeingObtained(clientHello.ServerName)
					if err != nil {
						return nil, fmt.Errorf("get certificate %s: is certificate being obtained: %s", clientHello.ServerName, err)
					}
					if yes {
						time.Sleep(100 * time.Millisecond)
						continue
					}
					obtainCertificate = i == 0
					break
				}

				if obtainCertificate {
					s.Logger.Debugf("get certificate: %s: obtaining certificate for domain", clientHello.ServerName)
					cert, err := s.CertificateService.ObtainCertificate(clientHello.ServerName)
					if err != nil {
						return nil, fmt.Errorf("get certificate %s: obtain certificate: %s", clientHello.ServerName, err)
					}
					c = &tls.Certificate{}
					*c, err = tls.X509KeyPair([]byte(cert.Cert), []byte(cert.Key))
					if err != nil {
						return nil, fmt.Errorf("get certificate: %s: tls X509KeyPair: %s", clientHello.ServerName, err)
					}
					// Clean cached empty certificate.
					s.certificateCache.InvalidateCertificate(clientHello.ServerName)
				} else {
					c, err = s.certificateCache.Certificate(clientHello.ServerName)
					if err != nil {
						return nil, fmt.Errorf("get certificate: %s: certificate cache: %s", clientHello.ServerName, err)
					}
				}
			}
		case nil:
		default:
			return nil, fmt.Errorf("get certificate: %s: certificate cache: %s", clientHello.ServerName, err)
		}
		if c != nil {
			return c, nil
		}
		if len(tlsConfig.NameToCertificate) != 0 {
			name := strings.ToLower(clientHello.ServerName)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}

			if cert, ok := tlsConfig.NameToCertificate[name]; ok {
				return cert, nil
			}

			labels := strings.Split(name, ".")
			for i := range labels {
				labels[i] = "*"
				candidate := strings.Join(labels, ".")
				if cert, ok := tlsConfig.NameToCertificate[candidate]; ok {
					return cert, nil
				}
			}
		}
		return nil, fmt.Errorf("get certificate: %s: certificate not found", clientHello.ServerName)
	}

	if s.TLSCert != "" && s.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(s.TLSCert, s.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load certificates: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}

	return
}

// Serve starts servers.
func (s *Server) Serve() error {
	return s.servers.Serve()
}

// Shutdown gracefully terminates servers.
func (s *Server) Shutdown(ctx context.Context) {
	s.servers.Shutdown(ctx)
}
