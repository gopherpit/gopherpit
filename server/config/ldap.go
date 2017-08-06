// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"time"

	"resenje.org/marshal"
)

// LDAPOptions defines parameters for LDAP authentication.
type LDAPOptions struct {
	Enabled              bool             `json:"enabled" yaml:"enabled" envconfig:"ENABLED"`
	Host                 string           `json:"host" yaml:"host" envconfig:"HOST"`
	Port                 uint             `json:"port" yaml:"port" envconfig:"PORT"`
	Secure               bool             `json:"secure" yaml:"secure" envconfig:"SECURE"`
	Username             string           `json:"username" yaml:"username" envconfig:"USERNAME"`
	Password             string           `json:"password" yaml:"password" envconfig:"PASSWORD"`
	DN                   string           `json:"dn" yaml:"dn" envconfig:"DN"`
	DNUsers              string           `json:"dn-users" yaml:"dn-users" envconfig:"DN_USERS"`
	DNGroups             string           `json:"dn-groups" yaml:"dn-groups" envconfig:"DN_GROUPS"`
	AttributeUsername    string           `json:"attribute-username" yaml:"attribute-username" envconfig:"ATTRIBUTE_USERNAME"`
	AttributeName        string           `json:"attribute-name" yaml:"attribute-name" envconfig:"ATTRIBUTE_NAME"`
	AttributeEmail       string           `json:"attribute-email" yaml:"attribute-email" envconfig:"ATTRIBUTE_EMAIL"`
	AttributeGroupID     string           `json:"attribute-group-id" yaml:"attribute-group-id" envconfig:"ATTRIBUTE_GROUP_ID"`
	AttributeGroupMember string           `json:"attribute-group-member" yaml:"attribute-group-member" envconfig:"ATTRIBUTE_GROUP_MEMBER"`
	Groups               []string         `json:"groups" yaml:"groups" envconfig:"GROUPS"`
	MaxConnections       int              `json:"max-connections" yaml:"max-connections" envconfig:"MAX_CONNECTIONS"`
	Timeout              marshal.Duration `json:"timeout" yaml:"timeout" envconfig:"TIMEOUT"`
}

// NewLDAPOptions initializes LDAPOptions with default values.
func NewLDAPOptions() *LDAPOptions {
	return &LDAPOptions{
		Enabled:              false,
		Host:                 "localhost",
		Port:                 636,
		Secure:               true,
		Username:             "",
		Password:             "",
		DN:                   "dc=example,dc=com",
		DNUsers:              "ou=Users",
		DNGroups:             "ou=Groups",
		AttributeUsername:    "uid",
		AttributeName:        "cn",
		AttributeEmail:       "mail",
		AttributeGroupID:     "cn",
		AttributeGroupMember: "memberUid",
		Groups:               []string{},
		MaxConnections:       16,
		Timeout:              marshal.Duration(10 * time.Second),
	}
}

// VerifyAndPrepare implements application.Options interface.
func (o *LDAPOptions) VerifyAndPrepare() error { return nil }
