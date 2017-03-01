// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server // import "gopherpit.com/gopherpit/server"

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"resenje.org/email"
	"resenje.org/httputils"
	"resenje.org/httputils/file-server"
	"resenje.org/httputils/log/access"
	"resenje.org/logging"
	"resenje.org/recovery"

	"gopherpit.com/gopherpit/pkg/certificate-cache"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/notification"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/session"
	"gopherpit.com/gopherpit/services/user"
)

// Server contains all required properties, services and functions
// to provide core functionality.
type Server struct {
	Options

	logger              *logging.Logger
	auditLogger         *logging.Logger
	packageAccessLogger *logging.Logger

	handler         http.Handler
	internalHandler http.Handler

	startTime    time.Time
	assetsServer *fileServer.Server

	certificateCache certificateCache.Cache

	salt []byte

	servers []*http.Server

	templates map[string]*template.Template
}

// Options structure contains optional properties for the Server.
type Options struct {
	Name                    string
	Version                 string
	BuildInfo               string
	Brand                   string
	Domain                  string
	RedirectToHTTPS         bool
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
	TLSEnabled              bool

	EmailService    email.Service
	RecoveryService recovery.Service

	SessionService      session.Service
	UserService         user.Service
	NotificationService notification.Service
	CertificateService  certificate.Service
	PackagesService     packages.Service
}

// NewServer creates a new instance of Server with HTTP handlers.
func NewServer(o Options) (s *Server, err error) {
	// Initialize server
	if o.Name == "" {
		o.Name = "server"
	}
	if o.Version == "" {
		o.Version = "0"
	}
	if o.VerificationSubdomain == "" {
		o.VerificationSubdomain = "_" + s.Name
	}
	logger, err := logging.GetLogger("default")
	if err != nil {
		err = fmt.Errorf("get default logger: %s", err)
		return
	}
	auditLogger, err := logging.GetLogger("audit")
	if err != nil {
		logger.Warningf("get audit logger: %s", err)
	}
	packageAccessLogger, err := logging.GetLogger("package-access")
	if err != nil {
		logger.Warningf("get package access logger: %s", err)
	}
	s = &Server{
		Options:             o,
		logger:              logger,
		auditLogger:         auditLogger,
		packageAccessLogger: packageAccessLogger,
		certificateCache:    certificateCache.NewCache(o.CertificateService, 15*time.Minute, time.Minute),
		startTime:           time.Now(),
		templates:           map[string]*template.Template{},
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
		logger.Infof("saving new salt to file %s", saltFilename)
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
		"asset":           s.assetFunc,
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

	s.assetsServer.NotFoundHandler = http.HandlerFunc(s.htmlNotFoundHandler)
	s.assetsServer.ForbiddenHandler = http.HandlerFunc(s.htmlForbiddenHandler)
	s.assetsServer.InternalServerErrorHandler = http.HandlerFunc(s.htmlInternalServerErrorHandler)

	accessLogHandler := func(h http.Handler) http.Handler {
		logger, err := logging.GetLogger("access")
		if err != nil {
			panic(fmt.Sprintf("get access logger: %s", err))
		}
		return accessLog.NewHandler(h, logger)
	}

	//
	// Top level router
	//
	baseRouter := http.NewServeMux()

	//
	// Assets handler
	//
	baseRouter.Handle("/assets/", chainHandlers(
		handlers.CompressHandler,
		s.htmlRecoveryHandler,
		accessLogHandler,
		s.htmlMaxBodyBytesHandler,
		httputils.NoExpireHeadersHandler,
		finalHandler(s.assetsServer),
	))

	//
	// Frontend router
	//
	frontendRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/", chainHandlers(
		handlers.CompressHandler,
		s.htmlRecoveryHandler,
		accessLogHandler,
		s.htmlMaintenanceHandler,
		s.htmlMaxBodyBytesHandler,
		s.acmeUserHandler,
		finalHandler(frontendRouter),
	))
	// Frontend routes start
	frontendRouter.NotFoundHandler = chainHandlers(
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, map[string]string{
				"Cache-Control": "no-cache",
			})
		},
		func(h http.Handler) http.Handler {
			return httputils.NewStaticFilesHandler(h, "/", http.Dir(o.StaticDir))
		},
		finalHandlerFunc(s.htmlNotFoundHandler),
	)
	frontendRouter.Handle("/", s.htmlLoginAltHandler(
		chainHandlers(
			s.htmlValidatedEmailRequiredHandler,
			finalHandlerFunc(s.dashboardHandler),
		),
		chainHandlers(
			s.generateAntiXSRFCookieHandler,
			finalHandlerFunc(s.landingPageHandler),
		),
	))
	frontendRouter.Handle("/about", http.HandlerFunc(s.aboutHandler))
	frontendRouter.Handle("/license", http.HandlerFunc(s.licenseHandler))
	frontendRouter.Handle("/contact", chainHandlers(
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.contactHandler),
	))
	frontendRouter.Handle("/login", chainHandlers(
		s.htmlLoginRequiredHandler,
		finalHandler(http.RedirectHandler("/", http.StatusSeeOther)),
	))
	frontendRouter.Handle("/logout", http.HandlerFunc(s.logoutHandler))
	frontendRouter.Handle("/registration", s.htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		chainHandlers(
			s.generateAntiXSRFCookieHandler,
			finalHandlerFunc(s.registrationHandler),
		),
	))
	frontendRouter.Handle("/password-reset", s.htmlLoginAltHandler(
		http.RedirectHandler("/", http.StatusSeeOther),
		chainHandlers(
			s.generateAntiXSRFCookieHandler,
			finalHandlerFunc(s.passwordResetTokenHandler),
		),
	))
	frontendRouter.Handle(`/password-reset/{token}`, chainHandlers(
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.passwordResetHandler),
	))
	frontendRouter.Handle(`/email/{token}`, chainHandlers(
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.publicEmailSettingsHandler),
	))
	frontendRouter.Handle(`/email-validation/{token}`, chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.emailValidationHandler),
	))
	frontendRouter.Handle("/settings", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.settingsHandler),
	))
	frontendRouter.Handle("/settings/email", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.settingsEmailHandler),
	))
	frontendRouter.Handle("/settings/notifications", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.settingsNotificationsHandler),
	))
	frontendRouter.Handle("/settings/password", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.settingsPasswordHandler),
	))
	frontendRouter.Handle("/settings/delete-account", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.settingsDeleteAccountHandler),
	))

	frontendRouter.Handle("/domain", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainAddHandler),
	))
	frontendRouter.Handle("/domain/{id}", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		finalHandlerFunc(s.domainPackagesHandler),
	))
	frontendRouter.Handle("/domain/{id}/settings", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainSettingsHandler),
	))
	frontendRouter.Handle("/domain/{id}/team", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainTeamHandler),
	))
	frontendRouter.Handle("/domain/{id}/changelog", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainChangelogHandler),
	))
	frontendRouter.Handle("/domain/{id}/user", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainDomainUserGrantHandler),
	))
	frontendRouter.Handle("/domain/{id}/user/{user-id}/revoke", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainDomainUserRevokeHandler),
	))
	frontendRouter.Handle("/domain/{id}/owner", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainDomainOwnerChangeHandler),
	))
	frontendRouter.Handle("/domain/{domain-id}/package", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainPackageEditHandler),
	))
	frontendRouter.Handle("/package/{package-id}", chainHandlers(
		s.htmlLoginRequiredHandler,
		s.htmlValidatedEmailRequiredHandler,
		s.generateAntiXSRFCookieHandler,
		finalHandlerFunc(s.domainPackageEditHandler),
	))
	frontendRouter.Handle("/user/{id}", chainHandlers(
		s.htmlLoginRequiredHandler,
		finalHandlerFunc(s.userPageHandler),
	))
	// Frontend routes end

	//
	// Frontend API ruter
	//
	frontendAPIRouter := mux.NewRouter().StrictSlash(true)
	baseRouter.Handle("/i/", chainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		accessLogHandler,
		s.jsonMaintenanceHandler,
		s.jsonAntiXSRFHandler,
		jsonMaxBodyBytesHandler,
		finalHandler(frontendAPIRouter),
	))
	// Frontend API routes start
	frontendAPIRouter.Handle("/i/auth", jsonMethodHandler{
		"POST":   http.HandlerFunc(s.authLoginFEAPIHandler),
		"DELETE": http.HandlerFunc(s.authLogoutFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/registration", jsonMethodHandler{
		"POST": http.HandlerFunc(s.registrationFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset-token", jsonMethodHandler{
		"POST": http.HandlerFunc(s.passwordResetTokenFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/password-reset", jsonMethodHandler{
		"POST": http.HandlerFunc(s.passwordResetFEAPIHandler),
	})
	frontendAPIRouter.Handle(`/i/email/opt-out/{token:\w{27,}}`, jsonMethodHandler{
		"POST":   http.HandlerFunc(s.emailOptOutFEAPIHandler),
		"DELETE": http.HandlerFunc(s.emailRemoveOptOutFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/contact", jsonMethodHandler{
		"POST": s.htmlLoginAltHandler(
			http.HandlerFunc(s.contactPrivateFEAPIHandler),
			http.HandlerFunc(s.contactFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/user", jsonMethodHandler{
		"POST": http.HandlerFunc(s.userFEAPIHandler),
	})
	frontendAPIRouter.Handle("/i/user/email", jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			finalHandlerFunc(s.userEmailFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/user/notifications", jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			finalHandlerFunc(s.userNotificationsSettingsFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/user/email/validation-email", jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			finalHandlerFunc(s.userSendEmailValidationEmailFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/user/password", jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			finalHandlerFunc(s.userPasswordFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/user/delete", jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			finalHandlerFunc(s.userDeleteFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle("/i/register-acme-user", jsonMethodHandler{
		"POST": http.HandlerFunc(s.registerACMEUserFEAPIHandler),
	})

	frontendAPIRouter.Handle(`/i/certificate/{id}`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.certificateFEAPIHandler),
		),
	})

	frontendAPIRouter.Handle(`/i/domain`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle(`/i/domain/{id}`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainFEAPIHandler),
		),
		"DELETE": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainDeleteFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle(`/i/domain/{id}/user`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainUserGrantFEAPIHandler),
		),
		"DELETE": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainUserRevokeFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle(`/i/domain/{id}/owner`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.domainOwnerChangeFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle(`/i/package`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.packageFEAPIHandler),
		),
	})
	frontendAPIRouter.Handle(`/i/package/{id}`, jsonMethodHandler{
		"POST": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.packageFEAPIHandler),
		),
		"DELETE": chainHandlers(
			s.jsonLoginRequiredHandler,
			s.jsonValidatedEmailRequiredHandler,
			finalHandlerFunc(s.packageDeleteFEAPIHandler),
		),
	})
	// Frontend API routes end

	//
	// Final handler
	//
	s.handler = chainHandlers(
		s.domainHandler,
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, o.Headers)
		},
		finalHandler(baseRouter),
	)

	//
	// Top level internal router
	//
	internalBaseRouter := http.NewServeMux()

	//
	// Internal router
	//
	internalRouter := http.NewServeMux()
	internalBaseRouter.Handle("/", chainHandlers(
		handlers.CompressHandler,
		httputils.NoCacheHeadersHandler,
		finalHandler(internalRouter),
	))
	internalRouter.Handle("/", http.HandlerFunc(textNotFoundHandler))
	internalRouter.Handle("/status", http.HandlerFunc(s.statusHandler))
	internalRouter.Handle("/data", http.HandlerFunc(s.dataDumpHandler))

	internalRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	internalRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	internalRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	internalRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	internalRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	//
	// Internal API router
	//
	internalAPIRouter := http.NewServeMux()
	internalBaseRouter.Handle("/api/", chainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		httputils.NoCacheHeadersHandler,
		finalHandler(internalAPIRouter),
	))
	internalAPIRouter.Handle("/api/", http.HandlerFunc(jsonNotFoundHandler))
	internalAPIRouter.Handle("/api/status", http.HandlerFunc(s.statusAPIHandler))
	internalAPIRouter.Handle("/api/maintenance", jsonMethodHandler{
		"GET":    http.HandlerFunc(s.maintenanceStatusAPIHandler),
		"POST":   http.HandlerFunc(s.maintenanceOnAPIHandler),
		"DELETE": http.HandlerFunc(s.maintenanceOffAPIHandler),
	})

	//
	// Final internal handler
	//
	s.internalHandler = chainHandlers(
		func(h http.Handler) http.Handler {
			return httputils.NewSetHeadersHandler(h, o.Headers)
		},
		finalHandler(internalBaseRouter),
	)
	return
}

// ServeOptions structure contains options for HTTP servers
// when invoking Server.Serve.
type ServeOptions struct {
	Listen            string
	ListenTLS         string
	ListenInternal    string
	ListenInternalTLS string
	TLSKey            string
	TLSCert           string
}

// Serve starts HTTP servers based on provided ServeOptions properties.
func (s *Server) Serve(o ServeOptions) error {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS10,
		NextProtos:         []string{"h2"},
		ClientSessionCache: tls.NewLRUClientSessionCache(-1),
	}
	tlsConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// If ServerName is defined in Options as Domain and there is TLSCert in Options
		// use static configuration by returning nil or both cert and err
		if clientHello.ServerName == s.Domain && o.TLSCert != "" {
			return nil, nil
		}
		// Get certificate for this ServerName
		c, err := s.certificateCache.Certificate(clientHello.ServerName)
		switch err {
		case certificateCache.ErrCertificateNotFound, certificate.CertificateNotFound:
			// If ServerName is the same as configured domain or it's www subdomain
			// and tls listener is on https port 443, try to obtain the certificate.
			if strings.HasSuffix(o.ListenTLS, ":443") && (clientHello.ServerName == s.Domain || clientHello.ServerName == "www."+s.Domain) {
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
					s.logger.Debugf("get certificate: %s: obtaining certificate for domain", clientHello.ServerName)
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

	if o.TLSCert != "" && o.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(o.TLSCert, o.TLSKey)
		if err != nil {
			return fmt.Errorf("TLS Certificates: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}

	if o.ListenTLS != "" {
		ln, err := net.Listen("tcp", o.ListenTLS)
		if err != nil {
			return fmt.Errorf("listen tls '%v': %s", o.ListenTLS, err)
		}

		ln = &httputils.TLSListener{
			TCPListener: ln.(*net.TCPListener),
			TLSConfig:   tlsConfig,
		}

		server := &http.Server{
			Handler: s.nilRecoveryHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.TLS == nil {
					httputils.HTTPToHTTPSRedirectHandler(w, r)
					return
				}
				switch {
				case strings.HasSuffix(r.URL.Path, "/info/refs"):
					// Handle git refs info if git reference is set.
					if notFound := s.packageGitInfoRefsHandler(w, r); !notFound {
						return
					}
				case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
					// Handle git upload pack if git reference is set.
					if notFound := s.packageGitUploadPackHandler(w, r); !notFound {
						return
					}
				}
				// Handle go get domain/...
				if r.URL.Query().Get("go-get") == "1" {
					s.packageResolverHandler(w, r)
					return
				}
				s.handler.ServeHTTP(w, r)
			})),
			TLSConfig: tlsConfig,
		}
		s.servers = append(s.servers, server)

		go func() {
			defer s.RecoveryService.Recover()

			s.logger.Infof("TLS HTTP Listening on %v", o.ListenTLS)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				s.logger.Errorf("Serve TLS '%v': %s", o.ListenTLS, err)
			}
		}()
	}
	if o.Listen != "" {
		ln, err := net.Listen("tcp", o.Listen)
		if err != nil {
			return fmt.Errorf("listen '%v': %s", o.Listen, err)
		}

		var handler http.Handler

		if o.ListenTLS != "" && s.Domain != "" {
			// Initialize handler that will redirect http:// to https:// only if
			// certificate for configured domain or it's www subdomain is available.
			_, tlsPort, err := net.SplitHostPort(o.ListenTLS)
			if err != nil {
				return fmt.Errorf("invalid tls: %s", err)
			}
			if tlsPort == "443" {
				tlsPort = ""
			} else {
				tlsPort = ":" + tlsPort
			}
			wwwDomain := "www." + s.Domain
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				domain, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					domain = r.Host
				}
				if (domain == s.Domain || domain == wwwDomain) && !strings.HasPrefix(r.URL.Path, acmeURLPrefix) {
					c, _ := s.certificateCache.Certificate(s.Domain)
					if c != nil {
						http.Redirect(w, r, strings.Join([]string{"https://", s.Domain, tlsPort, r.RequestURI}, ""), http.StatusMovedPermanently)
						return
					}
				}
				s.handler.ServeHTTP(w, r)
			})
		} else {
			handler = s.handler
		}

		server := &http.Server{
			Addr: o.Listen,
			Handler: s.nilRecoveryHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/info/refs"):
					// Handle git refs info if git reference is set.
					if notFound := s.packageGitInfoRefsHandler(w, r); !notFound {
						return
					}
				case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
					// Handle git upload pack if git reference is set.
					if notFound := s.packageGitUploadPackHandler(w, r); !notFound {
						return
					}
				}
				// Handle go get domain/...
				if r.URL.Query().Get("go-get") == "1" {
					s.packageResolverHandler(w, r)
					return
				}
				handler.ServeHTTP(w, r)
			})),
		}
		s.servers = append(s.servers, server)

		go func() {
			defer s.RecoveryService.Recover()

			s.logger.Infof("Plain HTTP Listening on %v", o.Listen)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				s.logger.Errorf("Serve '%v': %s", o.Listen, err)
			}
		}()
	}

	if o.ListenInternalTLS != "" {
		ln, err := net.Listen("tcp", o.ListenInternalTLS)
		if err != nil {
			return fmt.Errorf("listen internal tls '%v': %s", o.ListenInternalTLS, err)
		}

		ln = &httputils.TLSListener{
			TCPListener: ln.(*net.TCPListener),
			TLSConfig:   tlsConfig,
		}

		server := &http.Server{
			Handler:   s.nilRecoveryHandler(s.internalHandler),
			TLSConfig: tlsConfig,
		}
		s.servers = append(s.servers, server)

		go func() {
			defer s.RecoveryService.Recover()

			s.logger.Infof("Internal TLS HTTP Listening on %v", o.ListenInternalTLS)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				s.logger.Errorf("Serve Internal TLS '%v': %s", o.ListenInternalTLS, err)
			}
		}()
	}

	if o.ListenInternal != "" {
		ln, err := net.Listen("tcp", o.ListenInternal)
		if err != nil {
			return fmt.Errorf("listen internal '%v': %s", o.ListenInternal, err)
		}

		server := &http.Server{
			Addr:    o.ListenInternal,
			Handler: s.nilRecoveryHandler(s.internalHandler),
		}
		s.servers = append(s.servers, server)

		go func() {
			defer s.RecoveryService.Recover()

			s.logger.Infof("Internal plain HTTP Listening on %v", o.ListenInternal)

			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				s.logger.Errorf("Serve internal '%v': %s", o.ListenInternal, err)
			}
		}()
	}
	return nil
}

// Version returns service version based on values from version and
// build information.
func (s Server) Version() string {
	if s.BuildInfo != "" {
		return fmt.Sprintf("%s-%s", s.Options.Version, s.BuildInfo)
	}
	return s.Options.Version
}

// Shutdown gracefully terminates HTTP servers.
func (s *Server) Shutdown(ctx context.Context) {
	s.logger.Debug("Shutting down HTTP servers")
	wg := sync.WaitGroup{}
	for _, server := range s.servers {
		wg.Add(1)
		go func(server *http.Server) {
			defer s.RecoveryService.Recover()
			defer wg.Done()

			if err := server.Shutdown(ctx); err != nil {
				s.logger.Errorf("Server shutdown: %s", err)
			}
		}(server)
	}
	wg.Wait()
}
