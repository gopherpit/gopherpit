// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gopherpit creates executable for the gopherpit program.
//
// Configuration loading, validation and initialization of all required
// services for server to function is done in this package. It integrates
// all server dependencies into a form of a single executable.

package main // import "gopherpit.com/gopherpit/cmd/gopherpit"

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"resenje.org/daemon"
	"resenje.org/email"
	"resenje.org/httputils/client/api"
	"resenje.org/httputils/client/http"
	"resenje.org/logging"
	"resenje.org/recovery"

	"gopherpit.com/gopherpit/pkg/service"
	"gopherpit.com/gopherpit/server"
	"gopherpit.com/gopherpit/server/config"
	"gopherpit.com/gopherpit/services/certificate"
	"gopherpit.com/gopherpit/services/certificate/bolt"
	"gopherpit.com/gopherpit/services/certificate/http"
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

var (
	// Command line flag set.
	cli = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Command line interface options.
	configDir = cli.String("config-dir", "", "Directory that contains configuration files.")
	debug     = cli.Bool("debug", false, "Debug mode.")
	help      = cli.Bool("h", false, "Show program usage.")

	// Usage function.
	usage = func() {
		fmt.Fprintf(os.Stderr, `USAGE

  %s [options...] [command]

  Executing the program without specifying a command will start a process in
  the foreground and log all messages to stderr.

COMMANDS

  daemon
    Start program in the background.

  stop
    Stop program that runs in the background.

  status
    Dispaly status of a running process.

  config
    Print configuration that program will load on start. This command is
    dependent of -config-dir option value.

  debug-dump
    Send to a running process USR1 signal to log debug information in the log.

OPTIONS

`, os.Args[0])
		cli.PrintDefaults()
	}
)

func main() {
	cli.Usage = usage

	// Parse cli arguments.
	cli.Parse(os.Args[1:])
	arg0 := cli.Arg(0)

	// Handle help option.
	if *help {
		cli.Usage()
		fmt.Fprintln(os.Stderr, `
COPYRIGHT

  Copyright (C) 2016 Janoš Guljaš <janos@resenje.org>

  All rights reserved.
  Use of this source code is governed by a BSD-style
  license that can be found in the LICENSE file.
`)
		return
	}

	// Initialize configurations with default values.
	gopherpitOptions := config.NewGopherPitOptions()
	loggingOptions := config.NewLoggingOptions()
	emailOptions := config.NewEmailOptions()
	ldapOptions := config.NewLDAPOptions()
	sessionOptions := config.NewSessionOptions()
	userOptions := config.NewUserOptions()
	certificateOptions := config.NewCertificateOptions()
	servicesOptions := config.NewServicesOptions()
	// Make options list to be able to use them in config.Update and
	// config.Prepare.
	options := []config.Options{
		gopherpitOptions,
		loggingOptions,
		emailOptions,
		ldapOptions,
		sessionOptions,
		userOptions,
		certificateOptions,
		servicesOptions,
	}

	if *configDir == "" {
		*configDir = os.Getenv(strings.ToUpper(config.Name) + "_CONFIGDIR")
	}
	if *configDir == "" {
		*configDir = config.Dir
	}
	// Update options structures based on files in configDir and environment
	// variables.
	if err := config.Update(options, filepath.Join(config.BaseDir, "defaults"), *configDir); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Validate the command provided as first non-option argument.
	switch arg0 {
	case "":
		// No command is interpreted as starting the service in the foreground.

	case "daemon":
		// Daemon command is interpreted as starting the service in the
		// background. This command is verified at the end of this function.
		// This behavior provides functionality to validate configuration,
		// prepare storage and return errors before the process is daemonized.

	case "stop":
		err := service.StopDaemon(daemon.Daemon{
			PidFileName: gopherpitOptions.PidFileName,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Println("Stopped")
		return

	case "status":
		// Use daemon.Daemon to obtain status information and print it.
		pid, err := (&daemon.Daemon{
			PidFileName: gopherpitOptions.PidFileName,
		}).Status()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not running:", err)
			os.Exit(2)
		}
		fmt.Println("Running: PID", pid)
		return

	case "debug-dump":
		// Send SIGUSR1 signal to a daemonized process.
		// Service is able to receive the signal and dump debugging
		// information to files or stderr.
		err := (&daemon.Daemon{
			PidFileName: gopherpitOptions.PidFileName,
		}).Signal(syscall.SIGUSR1)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		return

	case "config":
		// Print loaded configuration.
		fmt.Printf("# gopherpit\n---\n%s\n", gopherpitOptions.String())
		fmt.Printf("# logging\n---\n%s\n", loggingOptions.String())
		fmt.Printf("# email\n---\n%s\n", emailOptions.String())
		fmt.Printf("# ldap\n---\n%s\n", ldapOptions.String())
		fmt.Printf("# session\n---\n%s\n", sessionOptions.String())
		fmt.Printf("# user\n---\n%s\n", userOptions.String())
		fmt.Printf("# certificate\n---\n%s\n", certificateOptions.String())
		fmt.Printf("# services\n---\n%s\n", servicesOptions.String())
		fmt.Printf("# config directories\n---\n- %s\n- %s\n", *configDir, filepath.Join(config.BaseDir, "defaults"))
		return

	default:
		// All other commands are invalid.
		fmt.Fprintln(os.Stderr, "unknown command:", arg0)
		cli.Usage()
		os.Exit(2)
	}

	// Continue to starting the service.

	// Verify options values and provide help and error message in case of
	// an error.
	if help, err := config.Verify(options); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if help != "" {
			fmt.Println()
			fmt.Println(help)
		}
		os.Exit(2)
	}
	// Execute prepare methods on options structures.
	// Usually it creates required directories of files.
	if err := config.Prepare(options); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Initialize the service with loaded options.
	s, err := service.NewService(
		config.Name,
		service.Options{
			HomeDir:                     gopherpitOptions.StorageDir,
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
			PidFileName:                 gopherpitOptions.PidFileName,
			PidFileMode:                 gopherpitOptions.PidFileMode.FileMode(),
			DaemonLogFileName:           loggingOptions.DaemonLogFileName,
			DaemonLogFileMode:           loggingOptions.DaemonLogFileMode.FileMode(),
		})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	logger, err := logging.GetLogger("default")
	if err != nil {
		return
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
		sessionService = httpSession.NewService(c)
	} else {
		db, err := boltSession.NewDB(filepath.Join(gopherpitOptions.StorageDir, "session.db"), gopherpitOptions.StorageFileMode.FileMode(), nil)
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
		userService = httpUser.NewService(c)
	} else {
		db, err := boltUser.NewDB(filepath.Join(gopherpitOptions.StorageDir, "user.db"), gopherpitOptions.StorageFileMode.FileMode(), nil)
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
		notificationService = httpNotification.NewService(c)
	} else {
		db, err := boltNotification.NewDB(filepath.Join(gopherpitOptions.StorageDir, "notification.db"), gopherpitOptions.StorageFileMode.FileMode(), nil)
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
		certificateService = httpCertificate.NewService(c)
	} else {
		db, err := boltCertificate.NewDB(filepath.Join(gopherpitOptions.StorageDir, "certificate.db"), gopherpitOptions.StorageFileMode.FileMode(), nil)
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
		packagesService = httpPackages.NewService(c)
	} else {
		db, err := boltPackages.NewDB(filepath.Join(gopherpitOptions.StorageDir, "packages.db"), gopherpitOptions.StorageFileMode.FileMode(), nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "packages service bolt database:", err)
			os.Exit(2)
		}
		changelog, err := boltPackages.NewChangelogPool(filepath.Join(gopherpitOptions.StorageDir, "changelog"), gopherpitOptions.StorageFileMode.FileMode(), nil)
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

	// Initialize server.
	srv, err := server.NewServer(
		server.Options{
			Name:                    config.Name,
			Version:                 config.Version,
			BuildInfo:               config.BuildInfo,
			Brand:                   gopherpitOptions.Brand,
			Domain:                  gopherpitOptions.Domain,
			Headers:                 gopherpitOptions.Headers,
			XSRFCookieName:          gopherpitOptions.XSRFCookieName,
			SessionCookieName:       gopherpitOptions.SessionCookieName,
			AssetsDir:               gopherpitOptions.AssetsDir,
			StaticDir:               gopherpitOptions.StaticDir,
			TemplatesDir:            gopherpitOptions.TemplatesDir,
			StorageDir:              gopherpitOptions.StorageDir,
			MaintenanceFilename:     gopherpitOptions.MaintenanceFilename,
			GoogleAnalyticsID:       gopherpitOptions.GoogleAnalyticsID,
			RememberMeDays:          userOptions.RememberMeDays,
			DefaultFrom:             emailOptions.DefaultFrom,
			ContactRecipientEmail:   gopherpitOptions.ContactRecipientEmail,
			ACMEDirectoryURL:        certificateOptions.DirectoryURL,
			ACMEDirectoryURLStaging: certificateOptions.DirectoryURLStaging,
			SkipDomainVerification:  gopherpitOptions.SkipDomainVerification,
			VerificationSubdomain:   gopherpitOptions.VerificationSubdomain,
			ForbiddenDomains:        gopherpitOptions.ForbiddenDomains,
			TLSEnabled:              gopherpitOptions.ListenTLS != "",

			EmailService:        *emailService,
			RecoveryService:     *recoveryService,
			SessionService:      sessionService,
			UserService:         userService,
			NotificationService: notificationService,
			CertificateService:  certificateService,
			PackagesService:     packagesService,
		},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Append Server functions.
	// All functions must be non-blocking or short-lived.
	// They will be executed in the same goroutine in the same order.
	s.Functions = append(s.Functions, func() error {
		return srv.Serve(server.ServeOptions{
			Listen:            gopherpitOptions.Listen,
			ListenTLS:         gopherpitOptions.ListenTLS,
			ListenInternal:    gopherpitOptions.ListenInternal,
			ListenInternalTLS: gopherpitOptions.ListenInternalTLS,
			TLSKey:            gopherpitOptions.TLSKey,
			TLSCert:           gopherpitOptions.TLSCert,
		})
	})
	if service, ok := sessionService.(*boltSession.Service); ok {
		// Start session cleanup.
		s.Functions = append(s.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := userService.(*boltUser.Service); ok {
		// Start user cleanup of email validations and password resets.
		s.Functions = append(s.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := notificationService.(*boltNotification.Service); ok {
		// Start celanup of expired email message IDs.
		s.Functions = append(s.Functions, func() error {
			return service.PeriodicCleanup()
		})
	}
	if service, ok := certificateService.(*boltCertificate.Service); ok {
		if gopherpitOptions.ListenTLS != "" || gopherpitOptions.ListenInternalTLS != "" {
			// Start renewal of certificates.
			s.Functions = append(s.Functions, service.PeriodicRenew)
		}
	}

	// Put the process in the background only if the Pid is not 1
	// (for example in docker) and the command is `daemon`.
	if syscall.Getpid() != 1 && arg0 == "daemon" {
		s.Daemonize()
	}

	// Finally start the server.
	// This is blocking function.
	if err := s.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}
}
