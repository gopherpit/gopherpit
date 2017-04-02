// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package certificateCache

import (
	"errors"
	"testing"
	"time"

	"gopherpit.com/gopherpit/services/certificate"
)

type CertificateGetter struct {
	getCount int
}

var errMock = errors.New("mocked error error")

func (g *CertificateGetter) Certificate(fqdn string) (c *certificate.Certificate, err error) {
	g.getCount++
	switch fqdn {
	case "invalid.gopherpit.com":
		return &certificate.Certificate{
			FQDN: fqdn,
		}, nil
	case "missing.gopherpit.com":
		return nil, certificate.ErrCertificateNotFound
	case "error.gopherpit.com":
		return nil, errMock
	case "nil.gopherpit.com":
		return nil, nil
	default:
		return &certificate.Certificate{
			FQDN: fqdn,
			Cert: `
-----BEGIN CERTIFICATE-----
MIICKjCCAZOgAwIBAgIJAIMSNhoBKZFaMA0GCSqGSIb3DQEBCwUAMC4xCzAJBgNV
BAYTAlVTMQswCQYDVQQIDAJDQTESMBAGA1UECgwJR29waGVyUGl0MB4XDTE2MTAy
NjE4NTg1MVoXDTI2MTAyNDE4NTg1MVowLjELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
AkNBMRIwEAYDVQQKDAlHb3BoZXJQaXQwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJ
AoGBAMGJE+nHSLxikzKG5ZniuFGed/uZwWA9EEOUE5MDmkZuUSnOCZ5v1v5rDRha
qW8rqTavbtW8bkhKKdMx5GnG3+6TTElgHYGYMDtbEBbTswx0+i9wOJXB11T7AQeu
dusElI0Gv0c5ss73emMNXUUUH9yQiVNrxYLDKDWQWyScQQTzAgMBAAGjUDBOMB0G
A1UdDgQWBBQqFuN3a+4dNTyQNzINs+as1LPJUzAfBgNVHSMEGDAWgBQqFuN3a+4d
NTyQNzINs+as1LPJUzAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4GBAFoQ
FI98+XEBw9fZLtTQy8Oc/NyyO6iWhntUKX7uzXgyWL8bD6gQEWFIqo8e+Rm8SRme
tMi8m5YerewsdKcNqnSononmdbEvpExp1byloBQkkbNkMZ8D8CrfBvw907TTdFEZ
EKQgSkR7QBLsu++nSYLjXcsWs3vRnLp5grSssCjh
-----END CERTIFICATE-----`,
			Key: `
-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDBiRPpx0i8YpMyhuWZ4rhRnnf7mcFgPRBDlBOTA5pGblEpzgme
b9b+aw0YWqlvK6k2r27VvG5ISinTMeRpxt/uk0xJYB2BmDA7WxAW07MMdPovcDiV
wddU+wEHrnbrBJSNBr9HObLO93pjDV1FFB/ckIlTa8WCwyg1kFsknEEE8wIDAQAB
AoGAPBzbto1Tpk/n8JW90yJ8pb1W/ysuyTmuR49C1TMVRDMXuqhojHGokbWmh54B
aqphEL9E6dZxWrrOau7gR4qiGulW0xY5u7CozGveMXhgUCn+ti3hGsq8wS6OiZkz
oVGmFBXInLk4ejFAYnWH1OBQoi0AzHo+eZL2niaex2mC+UECQQD3Cbc7J+q7Yrii
ZhD/+FxC7lK35ACyL9h0W3sOGpsRuYjb+9Jf4yvo8JKe/4bXQ3SUiVv1JYde0eBG
LE/Q1SvJAkEAyI55ogGWh9UK8BEhXJXoqJP786YEVjy9urcnarhwR0YxiyNn8yZE
0IfbHzVRejar7N7/n/ArdwxSNKQQzegQ2wJBAMWkp00TzZAoFpIPWNCCEsaVx+ZJ
62ikMOg+/H+3N5OBvgZKPfDrXokaWCQPSgFVfaMNFl5WrSxme6mI8D6jHkkCQFfa
+fN7KJsGO41go7GwRcQbV4KrVjkE0MRLWWwJsb23RRrDftToDbsf2GB6dd/ItVXF
dkt05UV4U0aWHHpmz4MCQQCoexlXW7ce+6hLlQafgsiY18WFw/uXoGxHzSlpLUby
viBngkOY/zwTS9mYvM8ixsj16b2WWzajtjhBtihs+tur
-----END RSA PRIVATE KEY-----`,
		}, nil
	}
}

func TestCache(t *testing.T) {
	cache := NewCache(&CertificateGetter{}, time.Minute, time.Second)

	c1, err := cache.Certificate("gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate first time: %s", err)
	}
	c2, err := cache.Certificate("gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate second time: %s", err)
	}
	if c1 != c2 {
		t.Errorf("first and second certificate are not the same")
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(&CertificateGetter{}, time.Second, time.Second)

	c1, err := cache.Certificate("gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate first time: %s", err)
	}
	time.Sleep(time.Second)
	c2, err := cache.Certificate("gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate second time: %s", err)
	}
	if c1 == c2 {
		t.Errorf("first and second certificate are the same, cache is not invalidated")
	}
}

func TestCacheForMissingCertificate(t *testing.T) {
	getter := &CertificateGetter{}
	cache := NewCache(getter, time.Minute, time.Second)

	_, err := cache.Certificate("missing.gopherpit.com")
	if err != certificate.ErrCertificateNotFound {
		t.Fatalf("get missing certificate first time: expected error %v, got %v", certificate.ErrCertificateNotFound, err)
	}
	_, err = cache.Certificate("missing.gopherpit.com")
	if err != ErrCertificateNotFound {
		t.Fatalf("get missing certificate second time: expected error %v, got %v", ErrCertificateNotFound, err)
	}
	if getter.getCount != 1 {
		t.Errorf("cache for missing certificate made %d requests to certificate.Getter instead of 1", getter.getCount)
	}
}

func TestCacheExpirationForMissingCertificate(t *testing.T) {
	getter := &CertificateGetter{}
	cache := NewCache(getter, time.Second, time.Second)

	_, err := cache.Certificate("missing.gopherpit.com")
	if err != certificate.ErrCertificateNotFound {
		t.Fatalf("get missing certificate first time: expected error %v, got %v", certificate.ErrCertificateNotFound, err)
	}
	time.Sleep(time.Second)
	_, err = cache.Certificate("missing.gopherpit.com")
	if err != certificate.ErrCertificateNotFound {
		t.Fatalf("get missing certificate second time: expected error %v, got %v", certificate.ErrCertificateNotFound, err)
	}
	if getter.getCount != 2 {
		t.Errorf("cache for missing certificate made %d requests to certificate.Getter instead of 2", getter.getCount)
	}
}

func TestInvalidCertificate(t *testing.T) {
	cache := NewCache(&CertificateGetter{}, time.Minute, time.Second)
	errorMessage := "tls X509KeyPair: invalid.gopherpit.com: tls: failed to find any PEM data in certificate input"
	_, err := cache.Certificate("invalid.gopherpit.com")
	if err.Error() != errorMessage {
		t.Fatalf("get invalid certificate: expected error message \"%s\", got \"%s\"", errorMessage, err)
	}
}

func TestNilCertificate(t *testing.T) {
	cache := NewCache(&CertificateGetter{}, time.Minute, time.Second)

	_, err := cache.Certificate("nil.gopherpit.com")
	if err != ErrCertificateNotFound {
		t.Fatalf("get nil certificate: expected error %v, got %v", ErrCertificateNotFound, err)
	}
}

func TestUnknownError(t *testing.T) {
	cache := NewCache(&CertificateGetter{}, time.Minute, time.Second)
	errorMessage := "certificate: error.gopherpit.com: mocked error error"
	_, err := cache.Certificate("error.gopherpit.com")
	if err.Error() != errorMessage {
		t.Fatalf("get error certificate: expected error message \"%s\", got \"%s\"", errorMessage, err)
	}
}

func TestCertificateCacheInvalidation(t *testing.T) {
	getter := &CertificateGetter{}
	cache := NewCache(getter, time.Second, time.Second)

	_, err := cache.Certificate("www.gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate first time: got error %v", err)
	}
	cache.InvalidateCertificate("www.gopherpit.com")
	_, err = cache.Certificate("www.gopherpit.com")
	if err != nil {
		t.Fatalf("get certificate second time: got error %v", err)
	}
	if getter.getCount != 2 {
		t.Errorf("cache for certificate invalidation made %d requests to certificate.Getter instead of 2", getter.getCount)
	}
}
