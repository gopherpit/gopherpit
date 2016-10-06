// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"resenje.org/marshal"
)

// LDAPOptions defines parameters for LDAP authentication.
type LDAPOptions struct {
	Enabled              bool             `json:"enabled" envconfig:"ENABLED"`
	Host                 string           `json:"host" envconfig:"HOST"`
	Port                 uint             `json:"port" envconfig:"PORT"`
	Secure               bool             `json:"secure" envconfig:"SECURE"`
	Username             string           `json:"username" envconfig:"USERNAME"`
	Password             string           `json:"password" envconfig:"PASSWORD"`
	DN                   string           `json:"dn" envconfig:"DN"`
	DNUsers              string           `json:"dn-users" envconfig:"DN_USERS"`
	DNGroups             string           `json:"dn-groups" envconfig:"DN_GROUPS"`
	AttributeUsername    string           `json:"attribute-username" envconfig:"ATTRIBUTE_USERNAME"`
	AttributeName        string           `json:"attribute-name" envconfig:"ATTRIBUTE_NAME"`
	AttributeEmail       string           `json:"attribute-email" envconfig:"ATTRIBUTE_EMAIL"`
	AttributeGroupID     string           `json:"attribute-group-id" envconfig:"ATTRIBUTE_GROUP_ID"`
	AttributeGroupMember string           `json:"attribute-group-member" envconfig:"ATTRIBUTE_GROUP_MEMBER"`
	Groups               []string         `json:"groups" envconfig:"GROUPS"`
	MaxConnections       int              `json:"max-connections" envconfig:"MAX_CONNECTIONS"`
	Timeout              marshal.Duration `json:"timeout" envconfig:"TIMEOUT"`
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

// Update updates options by loading ldap.json files from:
//  - defaults subdirectory of the directory where service executable is.
//  - configDir parameter
func (o *LDAPOptions) Update(configDir string) error {
	for _, dir := range []string{
		defaultsDir,
		configDir,
	} {
		f := filepath.Join(dir, "ldap.json")
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
	data, _ := json.MarshalIndent(o, "", "    ")
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
