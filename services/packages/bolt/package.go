// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltPackages

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/pkg/boltutils"

	"gopherpit.com/gopherpit/services/packages"
)

var (
	bucketNamePackages                   = []byte("Packages")
	bucketNameIndexPackageIDDomainID     = []byte("Index_PackageID_DomainID")
	bucketNameIndexDomainIDPathPackageID = []byte("Index_DomainID_Path_PackageID")
	bucketNameIndexDisabledPackageIDs    = []byte("Index_Disabled_PackageID_1")
)

type packageRecord struct {
	id          string
	DomainID    string       `json:"domain-id,omitempty"`
	Path        string       `json:"path,omitempty"`
	VCS         packages.VCS `json:"vcs,omitempty"`
	RepoRoot    string       `json:"repo-root,omitempty"`
	GoSource    string       `json:"go-source,omitempty"`
	RedirectURL string       `json:"redirect-url,omitempty"`
	Disabled    bool         `json:"disabled,omitempty"`
}

func (p packageRecord) export(tx *bolt.Tx) (pkg *packages.Package, err error) {
	pkg = &packages.Package{
		ID:          p.id,
		Path:        p.Path,
		VCS:         p.VCS,
		RepoRoot:    p.RepoRoot,
		GoSource:    p.GoSource,
		RedirectURL: p.RedirectURL,
		Disabled:    p.Disabled,
	}
	if tx != nil && p.DomainID != "" {
		d, err := getDomainRecordByID(tx, []byte(p.DomainID))
		if err != nil {
			return pkg, err
		}
		pkg.Domain = d.export()
	}
	return
}

func (p *packageRecord) update(tx *bolt.Tx, o *packages.PackageOptions) (changes packages.Changes, err error) {
	if o == nil {
		return
	}
	if o.Domain != nil {
		prevDomain := &domainRecord{}
		newDomain := &domainRecord{}
		if p.DomainID != "" {
			prevDomain, err = getDomainRecord(tx, []byte(p.DomainID))
			if err != nil {
				return changes, err
			}
		}
		if *o.Domain != "" {
			newDomain, err = getDomainRecord(tx, []byte(*o.Domain))
			if err != nil {
				return changes, err
			}
			p.DomainID = newDomain.id
		} else {
			p.DomainID = ""
		}
		if prevDomain.id != newDomain.id {
			changes = append(changes, packages.Change{
				Field: "domain-id",
				From:  stringToStringPtr(prevDomain.id),
				To:    stringToStringPtr(newDomain.id),
			})
			changes = append(changes, packages.Change{
				Field: "domain-fqdn",
				From:  stringToStringPtr(prevDomain.FQDN),
				To:    stringToStringPtr(newDomain.FQDN),
			})
		}
	}
	if o.Path != nil {
		if p.Path != *o.Path {
			changes = append(changes, packages.Change{
				Field: "path",
				From:  stringToStringPtr(p.Path),
				To:    stringToStringPtr(*o.Path),
			})
		}
		p.Path = *o.Path
	}
	if o.VCS != nil {
		if p.VCS != *o.VCS {
			changes = append(changes, packages.Change{
				Field: "vcs",
				From:  stringToStringPtr(string(p.VCS)),
				To:    stringToStringPtr(string(*o.VCS)),
			})
		}
		p.VCS = *o.VCS
	}
	if o.RepoRoot != nil {
		if p.RepoRoot != *o.RepoRoot {
			changes = append(changes, packages.Change{
				Field: "repo-root",
				From:  stringToStringPtr(p.RepoRoot),
				To:    stringToStringPtr(*o.RepoRoot),
			})
		}
		p.RepoRoot = *o.RepoRoot
	}
	if o.GoSource != nil {
		if p.GoSource != *o.GoSource {
			changes = append(changes, packages.Change{
				Field: "go-source",
				From:  stringToStringPtr(p.GoSource),
				To:    stringToStringPtr(*o.GoSource),
			})
		}
		p.GoSource = *o.GoSource
	}
	if o.RedirectURL != nil {
		if p.RedirectURL != *o.RedirectURL {
			changes = append(changes, packages.Change{
				Field: "redirect-url",
				From:  stringToStringPtr(p.RedirectURL),
				To:    stringToStringPtr(*o.RedirectURL),
			})
		}
		p.RedirectURL = *o.RedirectURL
	}
	if o.Disabled != nil {
		if p.Disabled != *o.Disabled {
			changes = append(changes, packages.Change{
				Field: "disabled",
				From:  boolPtrToStringPtr(&p.Disabled),
				To:    boolPtrToStringPtr(o.Disabled),
			})
		}
		p.Disabled = *o.Disabled
	}
	return
}

func getPackageRecord(tx *bolt.Tx, id []byte) (p *packageRecord, err error) {
	bucket := tx.Bucket(bucketNamePackages)
	if bucket == nil {
		err = packages.PackageNotFound
		return
	}
	data := bucket.Get(id)
	if data == nil {
		err = packages.PackageNotFound
		return
	}
	if err = json.Unmarshal(data, &p); err != nil {
		return
	}
	p.id = string(id)
	return
}

func getPackageIDByPath(tx *bolt.Tx, domainID, path []byte) (id []byte, err error) {
	bucket := tx.Bucket(bucketNameIndexDomainIDPathPackageID)
	if bucket == nil {
		err = packages.PackageNotFound
		return
	}
	bucket = bucket.Bucket(domainID)
	if bucket == nil {
		err = packages.PackageNotFound
		return
	}
	id = bucket.Get(path)
	if id == nil {
		err = packages.PackageNotFound
		return
	}
	return
}

func (p *packageRecord) save(tx *bolt.Tx) (err error) {
	p.Path = strings.TrimRight(strings.TrimSpace(p.Path), "/")
	p.RepoRoot = strings.TrimSpace(p.RepoRoot)
	// Required fields
	if p.DomainID == "" {
		return packages.PackageDomainRequired
	}
	if p.Path == "" {
		return packages.PackagePathRequired
	}
	if p.VCS == "" {
		return packages.PackageVCSRequired
	}
	if p.RepoRoot == "" {
		return packages.PackageRepoRootRequired
	}

	// existing package record
	ep := &packageRecord{}
	if p.id == "" {
		// Generate new id
		id, err := newPackageID(tx)
		if err != nil {
			return fmt.Errorf("generate unique ID: %s", err)
		}
		p.id = id
		// Address must be unique
		_, err = getPackageIDByPath(tx, []byte(p.DomainID), []byte(p.Path))
		switch err {
		case packages.PackageNotFound:
		case nil:
			return packages.PackageAlreadyExists
		default:
			return fmt.Errorf("get package id by path: %s", err)
		}
	} else {
		// Check if package with p.ID exists
		cp, err := getPackageRecord(tx, []byte(p.id))
		if err != nil {
			return fmt.Errorf("package record save get package record %s: %s", p.id, err)
		}
		if cp != nil {
			ep = cp
		}
	}

	id := []byte(p.id)
	var bucket *bolt.Bucket

	// Domain ID index
	if p.DomainID != ep.DomainID {
		if ep.DomainID != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexPackageIDDomainID,
				id,
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		if p.DomainID != "" {
			if err := boltutils.BoltDeepPut(
				tx,
				bucketNameIndexPackageIDDomainID,
				id,
				[]byte(p.DomainID),
			); err != nil {
				return fmt.Errorf("bolt deep put: %s", err)
			}
		}
	}

	// Domain ID and Path index
	if p.DomainID != ep.DomainID || p.Path != ep.Path {
		if ep.DomainID != "" && ep.Path != "" {
			if err := boltutils.BoltDeepDelete(
				tx,
				bucketNameIndexDomainIDPathPackageID,
				[]byte(ep.DomainID),
				[]byte(ep.Path),
			); err != nil {
				return fmt.Errorf("bolt deep delete: %s", err)
			}
		}
		if p.DomainID != "" && p.Path != "" {
			if err := boltutils.BoltDeepPut(
				tx,
				bucketNameIndexDomainIDPathPackageID,
				[]byte(p.DomainID),
				[]byte(p.Path),
				id,
			); err != nil {
				return fmt.Errorf("bolt deep put: %s", err)
			}
		}
	}

	// Disabled index
	if p.Disabled == false && ep.Disabled == true {
		bucket = tx.Bucket(bucketNameIndexDisabledPackageIDs)
		if bucket != nil {
			if err := bucket.Delete(id); err != nil {
				return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexDisabledPackageIDs, ep.id, err)
			}
		}
	}
	if p.Disabled == true && ep.Disabled == false {
		bucket = tx.Bucket(bucketNameIndexDisabledPackageIDs)
		if bucket != nil {
			if err := bucket.Put(id, flagBytes); err != nil {
				return fmt.Errorf("bucket(%s).Put(%s, %s) %s", bucketNameIndexDisabledPackageIDs, p.id, flagBytes, err)
			}
		}
	}

	// Save the package record data
	value, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}
	bucket, err = tx.CreateBucketIfNotExists(bucketNamePackages)
	if err != nil {
		return fmt.Errorf("CreateBucketIfNotExists(%s) %s", bucketNamePackages, err)
	}
	if err := bucket.Put(id, value); err != nil {
		return fmt.Errorf("bucket(%s).Put(%s) %s", bucketNamePackages, p.id, err)
	}

	return nil
}

func (p packageRecord) delete(tx *bolt.Tx) (err error) {
	id := []byte(p.id)

	// Domain ID indexes
	if p.DomainID != "" {
		if err := boltutils.BoltDeepDelete(
			tx,
			bucketNameIndexPackageIDDomainID,
			id,
		); err != nil {
			return fmt.Errorf("bolt deep delete: %s", err)
		}

		if err := boltutils.BoltDeepDelete(
			tx,
			bucketNameIndexDomainIDPathPackageID,
			[]byte(p.DomainID),
			[]byte(p.Path),
		); err != nil {
			return fmt.Errorf("bolt deep delete: %s", err)
		}
	}

	// Disabled index
	bucket := tx.Bucket(bucketNameIndexDisabledPackageIDs)
	if bucket != nil {
		if err := bucket.Delete(id); err != nil {
			return fmt.Errorf("bucket(%s).Delete(%s): %s", bucketNameIndexDisabledPackageIDs, p.id, err)
		}
	}

	// Package data
	bucket, err = tx.CreateBucketIfNotExists(bucketNamePackages)
	if err != nil {
		return
	}
	return bucket.Delete(id)
}

func newPackageID(tx *bolt.Tx) (id string, err error) {
	bp := make([]byte, 2)
	binary.LittleEndian.PutUint16(bp, uint16(os.Getpid()))
	br := make([]byte, 19)
	bt := make([]byte, 4)
	binary.LittleEndian.PutUint32(bt, uint32(time.Now().UTC().Unix()))
	bucket, err := tx.CreateBucketIfNotExists(bucketNamePackages)
	if err != nil {
		return
	}
	for i := 0; i < 100; i++ {
		_, err = rand.Read(br)
		if err != nil {
			return
		}
		b := append(bt, append(bp, br...)...)
		id = strings.TrimRight(base32Encoding.EncodeToString(b), "=")
		if v := bucket.Get([]byte(id)); v == nil {
			return
		}
	}
	return "", errors.New("unable to generate unique domain id")
}
