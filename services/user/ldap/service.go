// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ldapUser provides a Service that uses LDAP for user authentication.
package ldapUser // import "gopherpit.com/gopherpit/services/user/ldap"

import (
	"crypto/tls"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	ldap "gopkg.in/ldap.v2"
	"resenje.org/x/data-dump"

	"gopherpit.com/gopherpit/services/user"
)

var (
	emailRegex = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)
)

type ldapError string

func (l ldapError) Error() string {
	return string(l)
}

var (
	errUserSearchResults   = ldapError("user search returned no results")
	errUserUsernameMissing = ldapError("user username is missing")
	errUserGroupSearch     = ldapError("user not found in group search")
)

// Logger defines interface for logging messages with various severity levels.
type Logger interface {
	Debugf(format string, a ...interface{})
}

// Options are holds LDAP specific parameters.
type Options struct {
	Enabled              bool
	Host                 string
	Port                 uint
	Secure               bool
	Username             string
	Password             string
	DN                   string
	DNUsers              string
	DNGroups             string
	AttributeUsername    string
	AttributeName        string
	AttributeEmail       string
	AttributeGroupID     string
	AttributeGroupMember string
	Groups               []string
	MaxConnections       int
	Timeout              time.Duration
}

// Service encapsulates an instance of user.Service and Options to wrap
// Authenticate method.
type Service struct {
	user.Service
	Options

	logger Logger

	connections []*ldap.Conn
	mu          *sync.RWMutex
}

// NewService returns a new instance of Service.
func NewService(userService user.Service, logger Logger, o Options) *Service {
	return &Service{
		Service:     userService,
		Options:     o,
		logger:      logger,
		connections: []*ldap.Conn{},
		mu:          &sync.RWMutex{},
	}
}

// Authenticate authenticates a user over LDAP and if authentication fails
// it falls back to authentication by wrapped user service. If the user did not exist
// in wrapped user service, it will be created.
func (s *Service) Authenticate(ref, password string) (u *user.User, err error) {
	if !s.Enabled {
		return s.Service.Authenticate(ref, password)
	}
	u, err = s.User(ref)
	switch err {
	case user.ErrUserNotFound:
		if emailRegex.MatchString(ref) {
			u = &user.User{
				Email: ref,
			}
		} else {
			u = &user.User{
				Username: ref,
			}
		}
	case nil:
	default:
		return
	}
	newUserOptions, err := s.ldapAuth(u, password, "")
	if err != nil {
		switch err.(type) {
		case ldapError, *ldap.Error:
			s.logger.Debugf("ldap auth: %s: %s: continuing with alternative auth methods", ref, err)
			return s.Service.Authenticate(ref, password)
		}
		return
	}
	s.logger.Debugf("ldap auth: %s: authenticated", ref)
	if newUserOptions != nil {
		u, err = s.CreateUser(newUserOptions)
		if err != nil {
			err = fmt.Errorf("ldap auth: %s: create user: %s", ref, err)
			return
		}
		s.logger.Debugf("ldap auth: %s: new user created: %s", ref, u.ID)
	}
	return
}

func (s *Service) ldapConnection() (*ldap.Conn, error) {
	s.mu.RLock()
	idleCount := len(s.connections)
	s.mu.RUnlock()

	if idleCount == 0 {
		var l *ldap.Conn

		if s.Port <= 0 {
			return l, fmt.Errorf("invalid LDAP port %d", s.Port)
		}
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
		if s.Secure {
			var err error
			l, err = ldap.DialTLS("tcp", addr, &tls.Config{ServerName: s.Host})
			if err != nil {
				return l, err
			}
		} else {
			var err error
			l, err = ldap.Dial("tcp", addr)
			if err != nil {
				return l, err
			}
		}
		return l, nil
	}

	s.mu.Lock()
	l := s.connections[0]
	s.connections = s.connections[1:]
	s.mu.Unlock()

	return l, nil
}

func (s *Service) releaseLDAPConnection(l *ldap.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.connections) < s.MaxConnections {
		s.connections = append(s.connections, l)
		return
	}
	l.Close()
}

func (s Service) ldapConnectionsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.connections)
}

func (s *Service) ldapAuthSingle(u *user.User, password, group string) (newUserOptions *user.Options, err error) {
	var l *ldap.Conn
	l, err = s.ldapConnection()
	if err != nil {
		return
	}
	l.SetTimeout(s.Timeout)

	defer func() {
		if errt, ok := err.(*ldap.Error); ok && errt.ResultCode == 200 {
			return
		}
		s.releaseLDAPConnection(l)
	}()

	DNUsersPath := fmt.Sprintf("%s,%s", s.DNUsers, s.DN)

	if s.Username != "" && s.Password != "" {
		if _, err = l.SimpleBind(ldap.NewSimpleBindRequest(s.Username, s.Password, nil)); err != nil {
			return
		}
	}
	var bindRequest *ldap.SimpleBindRequest
	switch {
	case u.Username != "":
		bindRequest = ldap.NewSimpleBindRequest(fmt.Sprintf("%s=%s,%s", ldap.EscapeFilter(s.AttributeUsername), ldap.EscapeFilter(u.Username), DNUsersPath), password, nil)
	case u.Email != "":
		bindRequest = ldap.NewSimpleBindRequest(fmt.Sprintf("%s=%s,%s", ldap.EscapeFilter(s.AttributeEmail), ldap.EscapeFilter(u.Email), DNUsersPath), password, nil)
	default:
		err = errors.New("user has no username or email")
		return
	}
	if _, err = l.SimpleBind(bindRequest); err != nil {
		return
	}

	if u.ID == "" {
		filter := ""
		switch {
		case u.Username != "":
			filter = fmt.Sprintf("(%s=%s)", s.AttributeUsername, u.Username)
		case u.Email != "":
			filter = fmt.Sprintf("(%s=%s)", s.AttributeEmail, u.Email)
		default:
			err = errors.New("user has no username or email")
			return
		}
		search := ldap.NewSearchRequest(
			DNUsersPath,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			filter,
			[]string{s.AttributeUsername, s.AttributeName, s.AttributeEmail},
			nil)
		var results *ldap.SearchResult
		results, err = l.Search(search)
		if err != nil {
			return
		}
		if len(results.Entries) == 0 {
			err = errUserSearchResults
			return
		}
		u.Username = results.Entries[0].GetAttributeValue(s.AttributeUsername)
		u.Email = results.Entries[0].GetAttributeValue(s.AttributeEmail)
		u.Name = results.Entries[0].GetAttributeValue(s.AttributeName)
		newUserOptions = &user.Options{}
		newUserOptions.Username = &u.Username
		newUserOptions.Email = &u.Email
		newUserOptions.Name = &u.Name
	}

	if len(s.Groups) > 0 {
		if u.Username == "" {
			err = errUserUsernameMissing
			return
		}
		found := false
		for _, group := range s.Groups {
			LDAPDNGroup := fmt.Sprintf("%s=%s,%s,%s", ldap.EscapeFilter(s.AttributeGroupID), ldap.EscapeFilter(group), s.DNGroups, s.DN)
			search := ldap.NewSearchRequest(
				LDAPDNGroup,
				ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
				fmt.Sprintf("(%s=%s)", ldap.EscapeFilter(s.AttributeGroupMember), ldap.EscapeFilter(u.Username)),
				[]string{},
				nil)
			var results *ldap.SearchResult
			results, err = l.Search(search)
			if err != nil {
				if errt, ok := err.(*ldap.Error); ok && errt.ResultCode == 32 {
					continue
				}
				return
			}
			if len(results.Entries) > 0 {
				found = true
				break
			}
		}

		if !found {
			err = errUserGroupSearch
		}
	}
	return
}

func (s *Service) ldapAuth(u *user.User, password, group string) (newUserOptions *user.Options, err error) {
	i := 0
	for {
		i++
		newUserOptions, err = s.ldapAuthSingle(u, password, group)
		if errt, ok := err.(*ldap.Error); ok && errt.ResultCode == 200 {
			if i > s.ldapConnectionsCount()+1 {
				break
			}
			continue
		}
		break
	}
	return
}

// DataDump proxies encapsulated User service's DataDump method if it implements
// dataDump.Interface interface.
func (s Service) DataDump(ifModifiedSince *time.Time) (dump *dataDump.Dump, err error) {
	if dd, ok := s.Service.(dataDump.Interface); ok {
		return dd.DataDump(ifModifiedSince)
	}
	return
}
