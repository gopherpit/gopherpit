// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
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
		DN:                   "dc=my-organization,dc=com",
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

// Update updates options by loading ldap.json files.
func (o *LDAPOptions) Update(dirs ...string) error {
	for _, dir := range dirs {
		f := filepath.Join(dir, "ldap.yaml")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadYAML(f, o); err != nil {
				return fmt.Errorf("load yaml config: %s", err)
			}
		}
		f = filepath.Join(dir, "ldap.json")
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			if err := loadJSON(f, o); err != nil {
				return fmt.Errorf("load json config: %s", err)
			}
		}
	}
	if err := envconfig.Process(strings.Replace(Name, "-", "_", -1)+"_ldap", o); err != nil {
		return fmt.Errorf("load env valiables: %s", err)
	}
	return nil
}

// String returns a JSON representation of the options.
func (o *LDAPOptions) String() string {
	data, _ := yaml.Marshal(o)
	return string(data)
}

// Verify doesn't do anything, just provides method for Options interface.
func (o *LDAPOptions) Verify() (help string, err error) {
	return
}

// Prepare doesn't do anything, just provides method for Options interface.
func (o *LDAPOptions) Prepare() error {
	return nil
}
