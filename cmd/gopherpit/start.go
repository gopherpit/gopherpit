// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/throttled/throttled.v2/store/memstore"

	"resenje.org/email"
	"resenje.org/httputils/client/api"
	"resenje.org/httputils/client/http"
	"resenje.org/logging"
	"resenje.org/recovery"

	"gopherpit.com/gopherpit/pkg/application"
	"gopherpit.com/gopherpit/server"
	"gopherpit.com/gopherpit/server/config"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/certificate/bolt"
	"gopherpit.com/gopherpit/services/certificate/http"
	"gopherpit.com/gopherpit/services/gcrastore"
	"gopherpit.com/gopherpit/services/gcrastore/http"
	"gopherpit.com/gopherpit/services/key"
	"gopherpit.com/gopherpit/services/key/bolt"
	"gopherpit.com/gopherpit/services/key/http"
	"gopherpit.com/gopherpit/services/notification"
	"gopherpit.com/gopherpit/services/notification/bolt"
	"gopherpit.com/gopherpit/services/notification/http"
	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/packages/bolt"
	"gopherpit.com/gopherpit/services/packages/http"
	"gopherpit.com/gopherpit/services/session"
	"gopherpit.com/gopherpit/services/session/bolt"
	"gopherpit.com/gopherpit/services/session/http"
	"gopherpit.com/gopherpit/services/user"
	"gopherpit.com/gopherpit/services/user/bolt"
	"gopherpit.com/gopherpit/services/user/http"
	"gopherpit.com/gopherpit/services/user/ldap"
)

func startCmd(daemon bool) {
	// Initialize the application with loaded options.
	app, err := application.NewApp(
		config.Name,
		application.Options{
			HomeDir:                     options.StorageDir,
			LogDir:                      loggingOptions.LogDir,
			LogLevel:                    loggingOptions.LogLevel,
			LogFileMode:                 loggingOptions.LogFileMode.FileMode(),
			LogDirectoryMode:            loggingOptions.LogDirectoryMode.FileMode(),
			SyslogFacility:              loggingOptions.SyslogFacility,
			SyslogTag:                   loggingOptions.SyslogTag,
			SyslogNetwork:               loggingOptions.SyslogNetwork,
			SyslogAddress:               loggingOptions.SyslogAddress,
			AccessLogLevel:              loggingOptions.AccessLogLevel,
			AccessSyslogFacility:        loggingOptions.AccessSyslogFacility,
			AccessSyslogTag:             loggingOptions.AccessSyslogTag,
			PackageAccessLogLevel:       loggingOptions.PackageAccessLogLevel,
			PackageAccessSyslogFacility: loggingOptions.PackageAccessSyslogFacility,
			PackageAccessSyslogTag:      loggingOptions.PackageAccessSyslogTag,
			AuditLogDisabled:            loggingOptions.AuditLogDisabled,
			AuditSyslogFacility:         loggingOptions.AuditSyslogFacility,
			AuditSyslogTag:              loggingOptions.AuditSyslogTag,
			ForceLogToStderr:            *debug,
			PidFileName:                 options.PidFileName,
			PidFileMode:                 options.PidFileMode.FileMode(),
			DaemonLogFileName:           loggingOptions.DaemonLogFileName,
			DaemonLogFileMode:           loggingOptions.DaemonLogFileMode.FileMode(),
		})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	logger, err := logging.GetLogger("default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: get default logger: %s", err)
		os.Exit(2)
	}
	accessLogger, err := logging.GetLogger("access")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: get access logger: %s", err)
		os.Exit(2)
	}
	auditLogger, err := logging.GetLogger("audit")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: get audit logger: %s", err)
		os.Exit(2)
	}
	packageAccessLogger, err := logging.GetLogger("package-access")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: get package access logger: %s", err)
		os.Exit(2)
	}

	// Initialize services required for server to function.

	// Email sending service consumes an external SMTP server.
	emailService := &email.Service{
		SMTPHost:        emailOptions.SMTPHost,
		SMTPPort:        emailOptions.SMTPPort,
		SMTPSkipVerify:  emailOptions.SMTPSkipVerify,
		SMTPIdentity:    emailOptions.SMTPIdentity,
		SMTPUsername:    emailOptions.SMTPUsername,
		SMTPPassword:    emailOptions.SMTPPassword,
		NotifyAddresses: emailOptions.NotifyAddresses,
		DefaultFrom:     emailOptions.DefaultFrom,
		SubjectPrefix:   emailOptions.SubjectPrefix,
	}
	// Recovery service provides unified way of logging and notifying
	// panic events.
	recoveryService := &recovery.Service{
		Version:   config.Version,
		BuildInfo: config.BuildInfo,
		LogFunc:   logger.Error,
		Notifier:  emailService,
	}

	// Session service can be configured to use different implementations.
	// If session endpoint in services options is not blank, use http service.
	var sessionService session.Service
	if servicesOptions.SessionEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.SessionEndpoint,
			Key:       servicesOptions.SessionKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.SessionOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.SessionOptions)
		}
		sessionService = httpSession.NewClient(c)
	} else {
		db, err := boltSession.NewDB(filepath.Join(options.StorageDir, "session.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "session service bolt database:", err)
			os.Exit(2)
		}
		sessionService = &boltSession.Service{
			DB:              db,
			DefaultLifetime: sessionOptions.DefaultLifetime.Duration(),
			CleanupPeriod:   sessionOptions.CleanupPeriod.Duration(),
			Logger:          logger,
		}
	}
	// User service can be configured to use different implementations.
	// If user endpoint in services options is not blank, use http service.
	var userService user.Service
	if servicesOptions.UserEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.UserEndpoint,
			Key:       servicesOptions.UserKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.UserOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.UserOptions)
		}
		userService = httpUser.NewClient(c)
	} else {
		db, err := boltUser.NewDB(filepath.Join(options.StorageDir, "user.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "user service bolt database:", err)
			os.Exit(2)
		}
		userService = &boltUser.Service{
			DB: db,
			PasswordNoReuseMonths: userOptions.PasswordNoReuseMonths,
			Logger:                logger,
		}
	}
	if ldapOptions.Enabled {
		userService = ldapUser.NewService(
			userService,
			logger,
			ldapUser.Options{
				Enabled:              ldapOptions.Enabled,
				Host:                 ldapOptions.Host,
				Port:                 ldapOptions.Port,
				Secure:               ldapOptions.Secure,
				Username:             ldapOptions.Username,
				Password:             ldapOptions.Password,
				DN:                   ldapOptions.DN,
				DNUsers:              ldapOptions.DNUsers,
				DNGroups:             ldapOptions.DNGroups,
				AttributeUsername:    ldapOptions.AttributeUsername,
				AttributeName:        ldapOptions.AttributeName,
				AttributeEmail:       ldapOptions.AttributeEmail,
				AttributeGroupID:     ldapOptions.AttributeGroupID,
				AttributeGroupMember: ldapOptions.AttributeGroupMember,
				Groups:               ldapOptions.Groups,
				MaxConnections:       ldapOptions.MaxConnections,
				Timeout:              ldapOptions.Timeout.Duration(),
			},
		)
	}
	var notificationService notification.Service
	if servicesOptions.NotificationEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.NotificationEndpoint,
			Key:       servicesOptions.NotificationKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.NotificationOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.NotificationOptions)
		}
		notificationService = httpNotification.NewClient(c)
	} else {
		db, err := boltNotification.NewDB(filepath.Join(options.StorageDir, "notification.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "notification service bolt database:", err)
			os.Exit(2)
		}
		notificationService = &boltNotification.Service{
			DB:             db,
			SMTPHost:       emailOptions.SMTPHost,
			SMTPPort:       emailOptions.SMTPPort,
			SMTPSkipVerify: emailOptions.SMTPSkipVerify,
			SMTPIdentity:   emailOptions.SMTPIdentity,
			SMTPUsername:   emailOptions.SMTPUsername,
			SMTPPassword:   emailOptions.SMTPPassword,
			CleanupPeriod:  4 * time.Hour,
			Logger:         logger,
		}
	}
	var certificateService certificate.Service
	if servicesOptions.CertificateEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.CertificateEndpoint,
			Key:       servicesOptions.CertificateKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.CertificateOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.CertificateOptions)
		}
		certificateService = httpCertificate.NewClient(c)
	} else {
		db, err := boltCertificate.NewDB(filepath.Join(options.StorageDir, "certificate.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "certificate service bolt database:", err)
			os.Exit(2)
		}
		certificateService = &boltCertificate.Service{
			DB: db,
			DefaultACMEDirectoryURL: certificateOptions.DirectoryURL,
			RenewPeriod:             certificateOptions.RenewPeriod.Duration(),
			RenewCheckPeriod:        certificateOptions.RenewCheckPeriod.Duration(),
			RecoveryService:         *recoveryService,
			Logger:                  logger,
		}
	}
	var packagesService packages.Service
	if servicesOptions.PackagesEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.PackagesEndpoint,
			Key:       servicesOptions.PackagesKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.PackagesOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.PackagesOptions)
		}
		packagesService = httpPackages.NewClient(c)
	} else {
		db, err := boltPackages.NewDB(filepath.Join(options.StorageDir, "packages.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "packages service bolt database:", err)
			os.Exit(2)
		}
		changelog, err := boltPackages.NewChangelogPool(filepath.Join(options.StorageDir, "changelog"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "packages service bolt changelog database pool:", err)
			os.Exit(2)
		}
		packagesService = &boltPackages.Service{
			DB:        db,
			Changelog: changelog,
			Logger:    logger,
		}
	}
	var keyService key.Service
	if servicesOptions.KeyEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.KeyEndpoint,
			Key:       servicesOptions.KeyKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.KeyOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.KeyOptions)
		}
		keyService = httpKey.NewClient(c)
	} else {
		db, err := boltKey.NewDB(filepath.Join(options.StorageDir, "key.db"), options.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "key service bolt database:", err)
			os.Exit(2)
		}
		keyService = &boltKey.Service{
			DB:     db,
			Logger: logger,
		}
	}
	var gcraStoreService gcrastore.Service
	if servicesOptions.GCRAStoreEndpoint != "" {
		c := &apiClient.Client{
			Endpoint:  servicesOptions.GCRAStoreEndpoint,
			Key:       servicesOptions.GCRAStoreKey,
			UserAgent: config.UserAgent,
		}
		if servicesOptions.GCRAStoreOptions != nil {
			c.HTTPClient = httpClient.New(servicesOptions.GCRAStoreOptions)
		}
		gcraStoreService = httpGCRAStore.NewClient(c)
	} else {
		gcraStoreService, err = memstore.New(65536)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gcra memstore:", err)
			os.Exit(2)
		}
	}

	// Initialize server.
	if err = server.Configure(
		server.Options{
			Name:                    config.Name,
			Version:                 config.Version,
			BuildInfo:               config.BuildInfo,
			Brand:                   options.Brand,
			Domain:                  options.Domain,
			Listen:                  options.Listen,
			ListenTLS:               options.ListenTLS,
			ListenInternal:          options.ListenInternal,
			ListenInternalTLS:       options.ListenInternalTLS,
			TLSKey:                  options.TLSKey,
			TLSCert:                 options.TLSCert,
			Headers:                 options.Headers,
			XSRFCookieName:          options.XSRFCookieName,
			SessionCookieName:       options.SessionCookieName,
			AssetsDir:               options.AssetsDir,
			StaticDir:               options.StaticDir,
			TemplatesDir:            options.TemplatesDir,
			StorageDir:              options.StorageDir,
			MaintenanceFilename:     options.MaintenanceFilename,
			GoogleAnalyticsID:       options.GoogleAnalyticsID,
			RememberMeDays:          userOptions.RememberMeDays,
			DefaultFrom:             emailOptions.DefaultFrom,
			ContactRecipientEmail:   options.ContactRecipientEmail,
			ACMEDirectoryURL:        certificateOptions.DirectoryURL,
			ACMEDirectoryURLStaging: certificateOptions.DirectoryURLStaging,
			SkipDomainVerification:  options.SkipDomainVerification,
			VerificationSubdomain:   options.VerificationSubdomain,
			TrustedDomains:          options.TrustedDomains,
			ForbiddenDomains:        options.ForbiddenDomains,
			APITrustedProxyCIDRs:    apiOptions.TrustedProxyCIDRs,
			APIProxyRealIPHeader:    apiOptions.ProxyRealIPHeader,
			APIHourlyRateLimit:      apiOptions.HourlyRateLimit,
			APIEnabled:              !apiOptions.Disabled,

			Logger:              logger,
			AccessLogger:        accessLogger,
			AuditLogger:         auditLogger,
			PackageAccessLogger: packageAccessLogger,

			EmailService:        *emailService,
			RecoveryService:     *recoveryService,
			SessionService:      sessionService,
			UserService:         userService,
			NotificationService: notificationService,
			CertificateService:  certificateService,
			PackagesService:     packagesService,
			KeyService:          keyService,
			GCRAStoreService:    gcraStoreService,
		},
	); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Append Server functions.
	// All functions must be non-blocking or short-lived.
	// They will be executed in the same goroutine in the same order.
	app.Functions = append(app.Functions, func() error {
		return server.Serve()
	})
	if service, ok := sessionService.(*boltSession.Service); ok {
		// Start session cleanup.
		app.Functions = append(app.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := userService.(*boltUser.Service); ok {
		// Start user cleanup of email validations and password resets.
		app.Functions = append(app.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := notificationService.(*boltNotification.Service); ok {
		// Start celanup of expired email message IDs.
		app.Functions = append(app.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := certificateService.(*boltCertificate.Service); ok {
		if options.ListenTLS != "" || options.ListenInternalTLS != "" {
			// Start renewal of certificates.
			app.Functions = append(app.Functions, service.PeriodicRenew)
		}
	}

	app.ShutdownFunc = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		server.Shutdown(ctx)
		cancel()
		return nil
	}

	// Put the process in the background only if the Pid is not 1
	// (for example in docker) and the command is `daemon`.
	if syscall.Getpid() != 1 && daemon {
		app.Daemonize()
	}

	// Finally start the server.
	// This is blocking function.
	if err := app.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}
