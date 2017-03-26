// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server // import "gopherpit.com/gopherpit/server"

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	throttled "gopkg.in/throttled/throttled.v2"
	"resenje.org/httputils"
	"resenje.org/httputils/file-server"
	"resenje.org/logging"
	"resenje.org/recovery"

	"gopherpit.com/gopherpit/pkg/certificate-cache"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/gcrastore"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/notification"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/session"
	"gopherpit.com/gopherpit/services/user"
)

// Only one server is needed, so it is global to this package.
var srv *server

// Server contains all required properties, services and functions
// to provide core functionality.
type server struct {
	Options

	handler         http.Handler
	internalHandler http.Handler

	startTime    time.Time
	assetsServer *fileServer.Server

	certificateCache certificateCache.Cache

	salt []byte

	tlsConfig        *tls.Config
	tlsEnabled       bool
	registerACMEUser bool

	servers []*http.Server

	port, portTLS, portInternal, portInternalTLS int

	templates map[string]*template.Template

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
	XSRFCookieName          string
	SessionCookieName       string
	AssetsDir               string
	StaticDir               string
	TemplatesDir            string
	StorageDir              string
	MaintenanceFilename     string
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

	EmailService    EmailService
	RecoveryService recovery.Service

	SessionService      session.Service
	UserService         user.Service
	NotificationService notification.Service
	CertificateService  certificate.Service
	PackagesService     packages.Service
	KeyService          key.Service
	GCRAStoreService    gcrastore.Service
}

// Configure initializes http server with provided options.
func Configure(o Options) (err error) {
	if o.Name == "" {
		o.Name = "server"
	}
	if o.Version == "" {
		o.Version = "0"
	}
	if o.VerificationSubdomain == "" {
		o.VerificationSubdomain = "_" + srv.Name
	}
	s := &server{
		Options:          o,
		certificateCache: certificateCache.NewCache(o.CertificateService, 15*time.Minute, time.Minute),
		startTime:        time.Now(),
		templates:        map[string]*template.Template{},
		tlsEnabled:       o.ListenTLS != "",
		registerACMEUser: o.ListenTLS != "",
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
	s.assetsServer = fileServer.New("/assets", s.AssetsDir, &fileServer.Options{
		Hasher:                fileServer.MD5Hasher{HashLength: 8},
		NoHashQueryStrings:    true,
		RedirectTrailingSlash: true,
		IndexPage:             "index.html",
	})

	// Parse static HTML documents used as loadable fragments in templates
	fragments := map[string]interface{}{}
	fragmentsPath := filepath.Join(s.TemplatesDir, "fragments")
	_, err = os.Stat(fragmentsPath)
	switch {
	case os.IsNotExist(err):
	case err == nil:
		if err = filepath.Walk(fragmentsPath, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(path, fragmentsPath+"/")
			name = strings.TrimSuffix(name, ".md")
			fragments[name] = markdown(data)
			return nil
		}); err != nil {
			return
		}
	default:
		return
	}

	// Populate template functions
	templateFunctions := template.FuncMap{
		"asset":           assetFunc,
		"relative_time":   relativeTimeFunc,
		"safehtml":        safeHTMLFunc,
		"year_range":      yearRangeFunc,
		"contains_string": containsStringFunc,
		"html_br":         htmlBrFunc,
		"map":             mapFunc,
		"base32encode":    base32encodeFunc,
		"is_gopherpit_domain": func(domain string) bool {
			return strings.HasSuffix(domain, "."+o.Domain)
		},
		"context": newContext(map[string]interface{}{
			"GoogleAnalyticsID": o.GoogleAnalyticsID,
			"AliasCNAME":        "alias." + o.Domain,
		}),
		"fragment": newContext(fragments),
	}

	// Parse template files
	for name, files := range templates {
		fs := []string{}
		for _, f := range files {
			fs = append(fs, filepath.Join(s.TemplatesDir, f))
		}
		s.templates[name], err = template.New("").Funcs(templateFunctions).Delims("[[", "]]").ParseFiles(fs...)
		if err != nil {
			return
		}
	}

	s.assetsServer.NotFoundHandler = http.HandlerFunc(htmlNotFoundHandler)
	s.assetsServer.ForbiddenHandler = http.HandlerFunc(htmlForbiddenHandler)
	s.assetsServer.InternalServerErrorHandler = http.HandlerFunc(htmlInternalServerErrorHandler)

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
			return fmt.Errorf("api rate limiter: %s", err)
		}
	}

	// Configure TLS
	s.tlsConfig = &tls.Config{
		MinVersion:         tls.VersionTLS10,
		NextProtos:         []string{"h2"},
		ClientSessionCache: tls.NewLRUClientSessionCache(-1),
	}
	s.tlsConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// If ServerName is defined in Options as Domain and there is TLSCert in Options
		// use static configuration by returning nil or both cert and err
		if clientHello.ServerName == srv.Domain && srv.TLSCert != "" {
			return nil, nil
		}
		// Get certificate for this ServerName
		c, err := srv.certificateCache.Certificate(clientHello.ServerName)
		switch err {
		case certificateCache.ErrCertificateNotFound, certificate.CertificateNotFound:
			// If ServerName is the same as configured domain or it's www subdomain
			// and tls listener is on https port 443, try to obtain the certificate.
			if strings.HasSuffix(srv.ListenTLS, ":443") && (clientHello.ServerName == srv.Domain || clientHello.ServerName == "www."+srv.Domain) {
				obtainCertificate := false
				// Check if there is not already a request for new certificate active.
				for i := 0; i < 50; i++ {
					yes, err := srv.CertificateService.IsCertificateBeingObtained(clientHello.ServerName)
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
					srv.Logger.Debugf("get certificate: %s: obtaining certificate for domain", clientHello.ServerName)
					cert, err := srv.CertificateService.ObtainCertificate(clientHello.ServerName)
					if err != nil {
						return nil, fmt.Errorf("get certificate %s: obtain certificate: %s", clientHello.ServerName, err)
					}
					c = &tls.Certificate{}
					*c, err = tls.X509KeyPair([]byte(cert.Cert), []byte(cert.Key))
					if err != nil {
						return nil, fmt.Errorf("get certificate: %s: tls X509KeyPair: %s", clientHello.ServerName, err)
					}
					// Clean cached empty certificate.
					srv.certificateCache.InvalidateCertificate(clientHello.ServerName)
				} else {
					c, err = srv.certificateCache.Certificate(clientHello.ServerName)
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
		if len(s.tlsConfig.NameToCertificate) != 0 {
			name := strings.ToLower(clientHello.ServerName)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}

			if cert, ok := s.tlsConfig.NameToCertificate[name]; ok {
				return cert, nil
			}

			labels := strings.Split(name, ".")
			for i := range labels {
				labels[i] = "*"
				candidate := strings.Join(labels, ".")
				if cert, ok := s.tlsConfig.NameToCertificate[candidate]; ok {
					return cert, nil
				}
			}
		}
		return nil, fmt.Errorf("get certificate: %s: certificate not found", clientHello.ServerName)
	}

	if s.TLSCert != "" && s.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(s.TLSCert, s.TLSKey)
		if err != nil {
			return fmt.Errorf("TLS Certificates: %s", err)
		}
		s.tlsConfig.Certificates = []tls.Certificate{cert}
		s.tlsConfig.BuildNameToCertificate()
	}

	// Set the global srv variable
	srv = s

	return
}

// Serve starts HTTP servers.
func Serve() error {
	if srv == nil {
		return errors.New("server not configured")
	}

	setupRouters()
	setupInternalRouters()

	if srv.ListenTLS != "" {
		ln, err := net.Listen("tcp", srv.ListenTLS)
		if err != nil {
			return fmt.Errorf("listen tls '%v': %s", srv.ListenTLS, err)
		}

		srv.portTLS = ln.Addr().(*net.TCPAddr).Port

		ln = &httputils.TLSListener{
			TCPListener: ln.(*net.TCPListener),
			TLSConfig:   srv.tlsConfig,
		}

		server := &http.Server{
			Handler: nilRecoveryHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.TLS == nil {
					httputils.HTTPToHTTPSRedirectHandler(w, r)
					return
				}
				switch {
				case strings.HasSuffix(r.URL.Path, "/info/refs"):
					// Handle git refs info if git reference is set.
					if notFound := packageGitInfoRefsHandler(w, r); !notFound {
						return
					}
				case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
					// Handle git upload pack if git reference is set.
					if notFound := packageGitUploadPackHandler(w, r); !notFound {
						return
					}
				}
				// Handle go get domain/...
				if r.URL.Query().Get("go-get") == "1" {
					packageResolverHandler(w, r)
					return
				}
				srv.handler.ServeHTTP(w, r)
			})),
			TLSConfig: srv.tlsConfig,
		}
		srv.servers = append(srv.servers, server)

		go func() {
			defer srv.RecoveryService.Recover()

			addr := net.JoinHostPort(ln.Addr().(*net.TCPAddr).IP.String(), strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))

			srv.Logger.Infof("TLS HTTP Listening on %v", addr)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				srv.Logger.Errorf("Serve TLS '%v': %s", addr, err)
			}
		}()
	}
	if srv.Listen != "" {
		ln, err := net.Listen("tcp", srv.Listen)
		if err != nil {
			return fmt.Errorf("listen '%v': %s", srv.Listen, err)
		}

		srv.port = ln.Addr().(*net.TCPAddr).Port

		var handler http.Handler

		if srv.ListenTLS != "" && srv.Domain != "" {
			// Initialize handler that will redirect http:// to https:// only if
			// certificate for configured domain or it's www subdomain is available.
			_, tlsPort, err := net.SplitHostPort(srv.ListenTLS)
			if err != nil {
				return fmt.Errorf("invalid tls: %s", err)
			}
			if tlsPort == "443" {
				tlsPort = ""
			} else {
				tlsPort = ":" + tlsPort
			}
			wwwDomain := "www." + srv.Domain
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				domain, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					domain = r.Host
				}
				if (domain == srv.Domain || domain == wwwDomain) && !strings.HasPrefix(r.URL.Path, acmeURLPrefix) {
					c, _ := srv.certificateCache.Certificate(srv.Domain)
					if c != nil {
						http.Redirect(w, r, strings.Join([]string{"https://", srv.Domain, tlsPort, r.RequestURI}, ""), http.StatusMovedPermanently)
						return
					}
				}
				srv.handler.ServeHTTP(w, r)
			})
		} else {
			handler = srv.handler
		}

		server := &http.Server{
			Addr: srv.Listen,
			Handler: nilRecoveryHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/info/refs"):
					// Handle git refs info if git reference is set.
					if notFound := packageGitInfoRefsHandler(w, r); !notFound {
						return
					}
				case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
					// Handle git upload pack if git reference is set.
					if notFound := packageGitUploadPackHandler(w, r); !notFound {
						return
					}
				}
				// Handle go get domain/...
				if r.URL.Query().Get("go-get") == "1" {
					packageResolverHandler(w, r)
					return
				}
				handler.ServeHTTP(w, r)
			})),
		}
		srv.servers = append(srv.servers, server)

		go func() {
			defer srv.RecoveryService.Recover()

			addr := net.JoinHostPort(ln.Addr().(*net.TCPAddr).IP.String(), strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))

			srv.Logger.Infof("Plain HTTP Listening on %v", addr)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				srv.Logger.Errorf("Serve '%v': %s", addr, err)
			}
		}()
	}

	if srv.ListenInternalTLS != "" {
		ln, err := net.Listen("tcp", srv.ListenInternalTLS)
		if err != nil {
			return fmt.Errorf("listen internal tls '%v': %s", srv.ListenInternalTLS, err)
		}

		srv.portInternalTLS = ln.Addr().(*net.TCPAddr).Port

		ln = &httputils.TLSListener{
			TCPListener: ln.(*net.TCPListener),
			TLSConfig:   srv.tlsConfig,
		}

		server := &http.Server{
			Handler:   nilRecoveryHandler(srv.internalHandler),
			TLSConfig: srv.tlsConfig,
		}
		srv.servers = append(srv.servers, server)

		go func() {
			defer srv.RecoveryService.Recover()

			addr := net.JoinHostPort(ln.Addr().(*net.TCPAddr).IP.String(), strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))

			srv.Logger.Infof("Internal TLS HTTP Listening on %v", addr)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				srv.Logger.Errorf("Serve Internal TLS '%v': %s", addr, err)
			}
		}()
	}

	if srv.ListenInternal != "" {
		ln, err := net.Listen("tcp", srv.ListenInternal)
		if err != nil {
			return fmt.Errorf("listen internal '%v': %s", srv.ListenInternal, err)
		}

		srv.portInternal = ln.Addr().(*net.TCPAddr).Port

		server := &http.Server{
			Addr:    srv.ListenInternal,
			Handler: nilRecoveryHandler(srv.internalHandler),
		}
		srv.servers = append(srv.servers, server)

		go func() {
			defer srv.RecoveryService.Recover()

			addr := net.JoinHostPort(ln.Addr().(*net.TCPAddr).IP.String(), strconv.Itoa(ln.Addr().(*net.TCPAddr).Port))

			srv.Logger.Infof("Internal plain HTTP Listening on %v", addr)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				srv.Logger.Errorf("Serve Internal '%v': %s", addr, err)
			}
		}()
	}
	return nil
}

// Shutdown gracefully terminates HTTP servers.
func Shutdown(ctx context.Context) {
	if srv == nil {
		return
	}

	srv.Logger.Debug("Shutting down HTTP servers")
	wg := sync.WaitGroup{}
	for _, server := range srv.servers {
		wg.Add(1)
		go func(server *http.Server) {
			defer srv.RecoveryService.Recover()
			defer wg.Done()

			if err := server.Shutdown(ctx); err != nil {
				srv.Logger.Errorf("Server shutdown: %s", err)
			}
		}(server)
	}
	wg.Wait()
}

// Version returns service version based on values from version and
// build information.
func version() string {
	if srv.BuildInfo != "" {
		return fmt.Sprintf("%s-%s", srv.Options.Version, srv.BuildInfo)
	}
	return srv.Options.Version
}
