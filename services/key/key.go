// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package key

import (
	"net"

	apiClient "resenje.org/httputils/client/api"
)

type Service interface {
	KeyByRef(ref string) (*Key, error)
	KeyBySecret(secret string) (*Key, error)
	CreateKey(ref string, o *Options) (*Key, error)
	UpdateKey(ref string, o *Options) (*Key, error)
	DeleteKey(ref string) error
	RegenerateSecret(ref string) (string, error)
	Keys(startID string, limit int) (KeysPage, error)
}

type Key struct {
	Secret             string      `json:"secret"`
	Ref                string      `json:"ref"`
	AuthorizedNetworks []net.IPNet `json:"authorized-networks,omitempty"`
}

type Options struct {
	AuthorizedNetworks *[]net.IPNet `json:"authorized-networks,omitempty"`
}

type Keys []Key

type KeysPage struct {
	Keys     Keys   `json:"keys"`
	Previous string `json:"previous,omitempty"`
	Next     string `json:"next,omitempty"`
	Count    int    `json:"count,omitempty"`
}

var ErrorRegistry = apiClient.NewMapErrorRegistry(nil, nil)

// Errors that are related to the Key Service.
var (
	KeyNotFound         = ErrorRegistry.MustAddMessageError(1000, "key not found")
	KeyRefAlreadyExists = ErrorRegistry.MustAddMessageError(1001, "key reference already exists")
	KeyRefRequired      = ErrorRegistry.MustAddMessageError(1002, "key reference required")
)
