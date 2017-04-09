// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltCertificate

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xenolf/lego/acme"
	jose "gopkg.in/square/go-jose.v1"

	"gopherpit.com/gopherpit/services/certificate"
)

type acmeUser struct {
	Email        string                     `json:"email"`
	Key          *rsa.PrivateKey            `json:"key"`
	Registration *acme.RegistrationResource `json:"registration"`
	DirectoryURL string                     `json:"directory-url"`
}

func registerACMEUser(directoryURL, email, filename string) (u *acmeUser, err error) {
	var privateKey *rsa.PrivateKey
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	u = &acmeUser{
		Email:        email,
		Key:          privateKey,
		DirectoryURL: directoryURL,
	}
	client, err := acme.NewClient(directoryURL, u, acme.RSA2048)
	if err != nil {
		return
	}
	u.Registration, err = client.Register()
	if err != nil {
		return
	}
	if err = client.AgreeToTOS(); err != nil {
		return
	}
	var data []byte
	data, err = json.MarshalIndent(u.export(), "", "    ")
	if err != nil {
		return
	}
	if err = ioutil.WriteFile(filename, data, 0600); err != nil {
		return
	}
	return
}

func (u acmeUser) GetEmail() string {
	return u.Email
}

func (u acmeUser) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}

func (u acmeUser) GetPrivateKey() crypto.PrivateKey {
	return crypto.PrivateKey(u.Key)
}

func (u acmeUser) export() *acmeUserRecord {
	r := &acmeUserRecord{
		Email:        u.Email,
		DirectoryURL: u.DirectoryURL,
	}
	if u.Key != nil {
		r.PrivateKey = x509.MarshalPKCS1PrivateKey(u.Key)
	}
	if u.Registration != nil {
		r.URL = u.Registration.URI
		r.NewAuthzURL = u.Registration.NewAuthzURL
		r.ID = u.Registration.Body.ID
	}
	return r
}

type acmeUserRecord struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	PrivateKey   []byte `json:"private-key"`
	URL          string `json:"url"`
	NewAuthzURL  string `json:"new-authz-url"`
	DirectoryURL string `json:"directory-url"`
}

func (u acmeUserRecord) export() *certificate.ACMEUser {
	return &certificate.ACMEUser{
		ID:           u.ID,
		Email:        u.Email,
		PrivateKey:   u.PrivateKey,
		URL:          u.URL,
		NewAuthzURL:  u.NewAuthzURL,
		DirectoryURL: u.DirectoryURL,
	}
}

func (s *Service) acmeUserFilename() string {
	if s.DB == nil {
		return ""
	}
	return filepath.Join(filepath.Dir(s.DB.Path()), "acme-user.json")
}

func (s *Service) acmeUser() (u *acmeUser, err error) {
	if s.acmeUserCache != nil {
		u = s.acmeUserCache
	}
	defer func() {
		if err == nil {
			s.acmeUserCache = u
		}
	}()
	filename := s.acmeUserFilename()
	_, err = os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = certificate.ErrACMEUserNotFound
			return
		}
		return
	}
	var data []byte
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	r := &acmeUserRecord{}
	if err = json.Unmarshal(data, r); err != nil {
		return
	}
	var key *rsa.PrivateKey
	key, err = x509.ParsePKCS1PrivateKey(r.PrivateKey)
	if err != nil {
		return
	}
	u = &acmeUser{
		Email: r.Email,
		Key:   key,
		Registration: &acme.RegistrationResource{
			Body: acme.Registration{
				Resource: "reg",
				ID:       r.ID,
				Key: jose.JsonWebKey{
					Key: &key.PublicKey,
				},
			},
			URI:         r.URL,
			NewAuthzURL: r.NewAuthzURL,
		},
		DirectoryURL: r.DirectoryURL,
	}
	return
}

type challengeProvider struct {
	s Service
}

func (p challengeProvider) Present(fqdn, token, keyAuth string) (err error) {
	_, err = p.s.UpdateACMEChallenge(fqdn, &certificate.ACMEChallengeOptions{
		Token:   &token,
		KeyAuth: &keyAuth,
	})
	return
}

func (p challengeProvider) CleanUp(fqdn, token, keyAuth string) (err error) {
	_, err = p.s.DeleteACMEChallenge(fqdn)
	return
}

func loadPEMCertificates(pemBlock []byte) ([][]byte, error) {
	var derBlock *pem.Block
	var certs [][]byte
	for {
		derBlock, pemBlock = pem.Decode(pemBlock)
		if derBlock == nil {
			break
		}
		if derBlock.Type != "CERTIFICATE" {
			return nil, certificate.ErrCertificateInvalid
		}
		certs = append(certs, derBlock.Bytes)
	}
	if len(certs) == 0 {
		return nil, certificate.ErrCertificateInvalid
	}
	return certs, nil
}
