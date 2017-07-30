// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpCertificate

import (
	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/certificate"
)

var errorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are returned from the HTTP server.
var (
	ErrCertificateNotFound   = errorRegistry.MustAddMessageError(1000, "Certificate Not Found")
	ErrCertificateInvalid    = errorRegistry.MustAddMessageError(1001, "Certificate Invalid")
	ErrFQDNMissing           = errorRegistry.MustAddMessageError(1100, "FQDN Missing")
	ErrFQDNInvalid           = errorRegistry.MustAddMessageError(1101, "FQDN Invalid")
	ErrFQDNExists            = errorRegistry.MustAddMessageError(1102, "FQDN Exists")
	ErrACMEUserNotFound      = errorRegistry.MustAddMessageError(1200, "ACME User Not Found")
	ErrACMEUserEmailInvalid  = errorRegistry.MustAddMessageError(1201, "ACME User Email Invalid")
	ErrACMEChallengeNotFound = errorRegistry.MustAddMessageError(1300, "ACME Challenge Not Found")
)

var errorMap = map[error]error{
	ErrCertificateNotFound:   certificate.ErrCertificateNotFound,
	ErrCertificateInvalid:    certificate.ErrCertificateInvalid,
	ErrFQDNMissing:           certificate.ErrFQDNMissing,
	ErrFQDNInvalid:           certificate.ErrFQDNInvalid,
	ErrFQDNExists:            certificate.ErrFQDNExists,
	ErrACMEUserNotFound:      certificate.ErrACMEUserNotFound,
	ErrACMEUserEmailInvalid:  certificate.ErrACMEUserEmailInvalid,
	ErrACMEChallengeNotFound: certificate.ErrACMEChallengeNotFound,
}

func getServiceError(err error) error {
	e, ok := errorMap[err]
	if ok {
		return e
	}
	return err
}
