// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltPackages

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"gopherpit.com/gopherpit/services/packages"
	"resenje.org/boltdbpool"
	"resenje.org/boltdbpool/timed"
)

var (
	mmapFlags int
)

// Service implements gopherpit.com/gopherpit/services/packages.Service interface.
type Service struct {
	DB        *bolt.DB
	Changelog *timed.Pool
}

// NewDB opens a new BoltDB database.
func NewDB(filename string, fileMode os.FileMode, boltOptions *bolt.Options) (db *bolt.DB, err error) {
	if boltOptions == nil {
		boltOptions = &bolt.Options{
			Timeout:   2 * time.Second,
			MmapFlags: mmapFlags,
		}
	}
	if fileMode == 0 {
		fileMode = 0640
	}
	db, err = bolt.Open(filename, fileMode, boltOptions)
	return
}

func NewChangelogPool(directory string, fileMode os.FileMode, boltOptions *bolt.Options) (*timed.Pool, error) {
	if boltOptions == nil {
		boltOptions = &bolt.Options{
			Timeout:   2 * time.Second,
			MmapFlags: mmapFlags,
		}
	}
	return timed.New(directory, timed.Daily, &boltdbpool.Options{
		ConnectionExpires: 5 * time.Minute,
		FileMode:          fileMode,
		BoltOptions:       boltOptions,
	})
}

func (s Service) Domain(ref string) (d *packages.Domain, err error) {
	var r *domainRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getDomainRecord(tx, []byte(ref))
		return
	}); err != nil {
		return
	}
	d = r.export()
	return
}

func (s Service) AddDomain(o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	r := domainRecord{}
	r.update(o)
	if err = s.DB.Update(func(tx *bolt.Tx) error {
		return r.save(tx)
	}); err != nil {
		return
	}
	d = r.export()

	err = s.newChangelogRecord(chagelogRecordData{
		domainID: d.ID,
		fqdn:     d.FQDN,
		userID:   byUserID,
		action:   packages.ActionAddDomain,
		changes: packages.Changes{
			packages.Change{
				Field: "fqdn",
				To:    o.FQDN,
			},
			packages.Change{
				Field: "owner-user-id",
				To:    o.OwnerUserID,
			},
			packages.Change{
				Field: "certificate-ignore",
				To:    boolPtrToStringPtr(o.CertificateIgnore),
			},
			packages.Change{
				Field: "disabled",
				To:    boolPtrToStringPtr(o.Disabled),
			},
		},
	})

	return
}

func (s Service) UpdateDomain(ref string, o *packages.DomainOptions, byUserID string) (d *packages.Domain, err error) {
	r := &domainRecord{}
	var crd chagelogRecordData
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getDomainRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		crd = chagelogRecordData{
			domainID: r.id,
			fqdn:     r.FQDN,
			userID:   byUserID,
			action:   packages.ActionUpdateDomain,
			changes:  r.update(o),
		}
		return r.save(tx)
	}); err != nil {
		return
	}
	d = r.export()

	if len(crd.changes) > 0 {
		err = s.newChangelogRecord(crd)
	}

	return
}

func (s Service) DeleteDomain(ref, byUserID string) (d *packages.Domain, err error) {
	r := &domainRecord{}
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err = getDomainRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		return r.delete(tx)
	}); err != nil {
		return
	}
	d = r.export()

	err = s.newChangelogRecord(chagelogRecordData{
		domainID: d.ID,
		fqdn:     d.FQDN,
		userID:   byUserID,
		action:   packages.ActionDeleteDomain,
	})

	return
}

func (s Service) DomainUsers(ref string) (users packages.DomainUsers, err error) {
	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		d, err := getDomainRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		users.OwnerUserID = d.OwnerUserID
		bucket := tx.Bucket(bucketNameIndexDomainIDUserIDs)
		if bucket != nil {
			bucket = bucket.Bucket([]byte(d.id))
			if bucket != nil {
				users.UserIDs = []string{}
				if err = bucket.ForEach(func(userID, _ []byte) error {
					id := string(userID)
					if id != d.OwnerUserID {
						users.UserIDs = append(users.UserIDs, id)
					}
					return nil
				}); err != nil {
					return
				}
			}
		}
		return
	})
	return
}

func (s Service) AddUserToDomain(ref, userID, byUserID string) (err error) {
	d := &domainRecord{}

	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		d, err = getDomainRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		if !d.isOwner(byUserID) {
			err = packages.Forbidden
			return
		}
		err = d.addUser(tx, []byte(userID), true)
		return
	}); err != nil {
		return
	}

	err = s.newChangelogRecord(chagelogRecordData{
		domainID: d.id,
		fqdn:     d.FQDN,
		userID:   byUserID,
		action:   packages.ActionDomainAddUser,
		changes: packages.Changes{
			packages.Change{
				Field: "id",
				To:    &userID,
			},
		},
	})

	return
}

func (s Service) RemoveUserFromDomain(ref, userID, byUserID string) (err error) {
	d := &domainRecord{}

	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		d, err = getDomainRecord(tx, []byte(ref))
		if err != nil {
			return
		}
		if !d.isOwner(byUserID) {
			err = packages.Forbidden
			return
		}
		err = d.removeUser(tx, []byte(userID))
		return
	}); err != nil {
		return
	}

	err = s.newChangelogRecord(chagelogRecordData{
		domainID: d.id,
		fqdn:     d.FQDN,
		userID:   byUserID,
		action:   packages.ActionDomainRemoveUser,
		changes: packages.Changes{
			packages.Change{
				Field: "id",
				To:    &userID,
			},
		},
	})

	return
}

func (s Service) Domains(startRef string, limit int) (page packages.DomainsPage, err error) {
	switch {
	case limit == 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	start := []byte(startRef)

	page = packages.DomainsPage{
		Domains: packages.Domains{},
	}
	err = s.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketNameIndexFQDNDomainID)
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		var k, v []byte
		if len(start) == 0 {
			k, v = c.First()
		} else {
			k, v = c.Seek(start)
			var prev, p []byte
			for i := 0; i < limit; i++ {
				p, _ = c.Prev()
				if p == nil {
					break
				}
				prev = p
			}
			page.Previous = string(prev)
			k, v = c.Seek(start)
		}
		var i int
		for i = 0; k != nil && i < limit; i++ {
			r, err := getDomainRecordByID(tx, v)
			if err != nil {
				return err
			}
			d := r.export()
			page.Domains = append(page.Domains, *d)
			k, v = c.Next()
		}
		page.Next = string(k)
		page.Count = i
		return nil
	})

	return
}

func (s Service) DomainsByUser(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	return s.domainsByUser(userID, startRef, limit, bucketNameIndexUserIDFQDNDomainID)
}

func (s Service) DomainsByOwner(userID, startRef string, limit int) (p packages.DomainsPage, err error) {
	return s.domainsByUser(userID, startRef, limit, bucketNameIndexOwnerUserIDFQDNDomainID)
}

func (s Service) domainsByUser(userID, startRef string, limit int, bucket []byte) (page packages.DomainsPage, err error) {
	switch {
	case limit == 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	start := []byte(startRef)

	page = packages.DomainsPage{
		Domains: packages.Domains{},
		UserID:  userID,
	}
	err = s.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucket)
		if bucket == nil {
			return nil
		}
		bucket = bucket.Bucket([]byte(userID))
		if bucket == nil {
			return packages.UserDoesNotExist
		}
		c := bucket.Cursor()
		var k, v []byte
		if len(start) == 0 {
			k, v = c.First()
		} else {
			k, v = c.Seek(start)
			var prev, p []byte
			for i := 0; i < limit; i++ {
				p, _ = c.Prev()
				if p == nil {
					break
				}
				prev = p
			}
			page.Previous = string(prev)
			k, v = c.Seek(start)
		}
		var i int
		for i = 0; k != nil && i < limit; i++ {
			r, err := getDomainRecordByID(tx, v)
			if err != nil {
				return err
			}
			d := r.export()
			page.Domains = append(page.Domains, *d)
			k, v = c.Next()
		}
		page.Next = string(k)
		page.Count = i
		return nil
	})

	return
}

func (s Service) Package(id string) (p *packages.Package, err error) {
	var r *packageRecord
	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		r, err = getPackageRecord(tx, []byte(id))
		if err != nil {
			return
		}
		p, err = r.export(tx)
		return
	}); err != nil {
		return
	}
	return
}

func (s Service) AddPackage(o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	if o.Domain == nil {
		err = packages.PackageDomainRequired
		return
	}

	var crd chagelogRecordData
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r := &packageRecord{}
		d, err := getDomainRecord(tx, []byte(*o.Domain))
		if err != nil {
			return err
		}
		if !d.isUser(tx, byUserID) {
			return packages.Forbidden
		}
		crd = chagelogRecordData{
			userID: byUserID,
			action: packages.ActionAddPackage,
		}
		crd.changes, err = r.update(tx, o)
		if err != nil {
			return err
		}
		if err = r.save(tx); err != nil {
			return
		}
		p, err = r.export(tx)
		crd.domainID = p.Domain.ID
		crd.fqdn = p.Domain.FQDN
		crd.packageID = p.ID
		crd.path = p.Path
		return
	}); err != nil {
		return
	}

	if len(crd.changes) > 0 {
		err = s.newChangelogRecord(crd)
	}

	return
}

func (s Service) UpdatePackage(id string, o *packages.PackageOptions, byUserID string) (p *packages.Package, err error) {
	var crd chagelogRecordData

	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		var r *packageRecord
		r, err = getPackageRecord(tx, []byte(id))
		if err != nil {
			return
		}

		d, err := getDomainRecordByID(tx, []byte(r.DomainID))
		if err != nil {
			return err
		}
		if !d.isUser(tx, byUserID) {
			return packages.Forbidden
		}
		crd = chagelogRecordData{
			userID: byUserID,
			action: packages.ActionUpdatePackage,
		}
		crd.changes, err = r.update(tx, o)
		if err != nil {
			return err
		}
		if err = r.save(tx); err != nil {
			return
		}
		p, err = r.export(tx)
		crd.domainID = p.Domain.ID
		crd.fqdn = p.Domain.FQDN
		crd.packageID = p.ID
		crd.path = p.Path
		return
	}); err != nil {
		return
	}

	if len(crd.changes) > 0 {
		err = s.newChangelogRecord(crd)
	}

	return
}

func (s Service) DeletePackage(id string, byUserID string) (p *packages.Package, err error) {
	if err = s.DB.Update(func(tx *bolt.Tx) (err error) {
		r, err := getPackageRecord(tx, []byte(id))
		if err != nil {
			return
		}
		d, err := getDomainRecordByID(tx, []byte(r.DomainID))
		if err != nil {
			return err
		}
		if !d.isUser(tx, byUserID) {
			err = packages.Forbidden
			return
		}
		if err = r.delete(tx); err != nil {
			return
		}
		p, err = r.export(tx)
		return
	}); err != nil {
		return
	}

	err = s.newChangelogRecord(chagelogRecordData{
		domainID:  p.Domain.ID,
		fqdn:      p.Domain.FQDN,
		packageID: p.ID,
		path:      p.Path,
		userID:    byUserID,
		action:    packages.ActionDeletePackage,
	})

	return
}

func (s Service) PackagesByDomain(domainRef, startName string, limit int) (page packages.PackagesPage, err error) {
	switch {
	case limit == 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	start := []byte(startName)

	page = packages.PackagesPage{
		Packages: packages.Packages{},
	}

	err = s.DB.View(func(tx *bolt.Tx) error {
		r, err := getDomainRecord(tx, []byte(domainRef))
		if err != nil {
			return err
		}
		page.Domain = r.export()
		bucket := tx.Bucket(bucketNameIndexDomainIDPathPackageID)
		if bucket == nil {
			return nil
		}
		bucket = bucket.Bucket([]byte(r.id))
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		var k, v []byte
		if len(start) == 0 {
			k, v = c.First()
		} else {
			k, v = c.Seek(start)
			var prev, p []byte
			for i := 0; i < limit; i++ {
				p, _ = c.Prev()
				if p == nil {
					break
				}
				prev = p
			}
			page.Previous = string(prev)
			k, v = c.Seek(start)
		}
		var i int
		for i = 0; k != nil && i < limit; i++ {
			d, err := getPackageRecord(tx, v)
			if err != nil {
				return err
			}
			p, err := d.export(nil)
			if err != nil {
				return err
			}
			page.Packages = append(page.Packages, *p)
			k, v = c.Next()
		}
		page.Next = string(k)
		page.Count = i
		return nil
	})

	return
}

func (s Service) ResolvePackage(path string) (resolution *packages.PackageResolution, err error) {
	var fqdn string
	i := strings.IndexRune(path, '/')
	if i < 0 {
		fqdn = path
	} else {
		fqdn = path[:i]
		path = path[i:]
	}

	err = s.DB.View(func(tx *bolt.Tx) (err error) {
		domainID, err := getDomainIDByFQDN(tx, []byte(fqdn))
		if err != nil {
			return
		}
		var packageID []byte
		parts := strings.Split(path, "/")
		for i := len(parts); i > 0; i-- {
			path = strings.Join(parts[:i], "/")
			packageID, err = getPackageIDByPath(tx, domainID, []byte(path))
			if err == packages.PackageNotFound {
				continue
			}
			break
		}
		p, err := getPackageRecord(tx, packageID)
		if err != nil {
			return
		}
		resolution = &packages.PackageResolution{}
		resolution.ImportPrefix = fqdn + path
		resolution.VCS = p.VCS
		resolution.RepoRoot = p.RepoRoot
		resolution.RefType = p.RefType
		resolution.RefName = p.RefName
		resolution.GoSource = p.GoSource
		resolution.RedirectURL = p.RedirectURL
		resolution.Disabled, err = isDomainDisabled(tx, domainID)
		if err != nil {
			return
		}
		if !resolution.Disabled {
			resolution.Disabled = p.Disabled
		}
		return
	})

	return
}

func (s Service) ChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	r := &changelogRecord{
		id: id,
	}

	t, err := r.getTime()
	if err != nil {
		return nil, err
	}

	c, err := s.Changelog.GetConnection(t)
	switch err {
	case timed.ErrUnknownDB:
		err = packages.ChangelogRecordNotFound
		return
	case nil:
		break
	default:
		return nil, err
	}
	defer c.Close()

	if err = c.DB.View(func(tx *bolt.Tx) (err error) {
		d := []byte(domainRef)
		r, err = getChangelogRecord(tx, d, []byte(id))
		if err == packages.DomainNotFound {
			d, err = getDomainIDByFQDN(tx, d)
			if err != nil {
				return
			}
			r, err = getChangelogRecord(tx, d, []byte(id))
		}
		return
	}); err != nil {
		return
	}
	record, err = r.export()

	return
}

func (s Service) DeleteChangelogRecord(domainRef, id string) (record *packages.ChangelogRecord, err error) {
	r := &changelogRecord{
		id: id,
	}

	t, err := r.getTime()
	if err != nil {
		return nil, err
	}

	c, err := s.Changelog.GetConnection(t)
	switch err {
	case timed.ErrUnknownDB:
		err = packages.ChangelogRecordNotFound
		return
	case nil:
		break
	default:
		return nil, err
	}
	defer c.Close()

	if err = c.DB.Update(func(tx *bolt.Tx) (err error) {
		d := []byte(domainRef)
		r, err = getChangelogRecord(tx, d, []byte(id))
		if err == packages.DomainNotFound {
			d, err = getDomainIDByFQDN(tx, d)
			if err != nil {
				return
			}
			r, err = getChangelogRecord(tx, d, []byte(id))
			if err != nil {
				return
			}
		}
		err = r.delete(tx)
		return
	}); err != nil {
		return
	}
	record, err = r.export()

	return
}

var timeLayouts = []string{
	"2006-01-02",
	"2006-01-02Z07:00",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02T15:04Z07:00",
	"2006-01-02 15:04Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
}

func (s Service) ChangelogForDomain(domainRef, start string, limit int) (page packages.Changelog, err error) {
	switch {
	case limit == 0:
		limit = 5
	case limit > 100:
		limit = 100
	}

	// default start time is the current time
	startTime := time.Now().UTC()
	if start != "" {
		for i, pattern := range timeLayouts {
			// if start query parameter is valid, use it as start time
			if startTime, err = time.ParseInLocation(pattern, start, time.UTC); err == nil {
				// as record order is reversed in time, adjust start time to make natural listing
				// this is related how boltdb cursor seek is working in reverse order
				switch i {
				case 0, 1:
					startTime = startTime.AddDate(0, 0, 1).Add(-time.Nanosecond)
				case 2, 3, 4, 5:
					startTime = startTime.Add(time.Minute).Add(-time.Nanosecond)
				case 6, 7, 8, 9:
					startTime = startTime.Add(time.Second).Add(-time.Nanosecond)
				}
				break
			}
		}
		if startTime.IsZero() {
			// set start time from start query parameter if it is zero
			startTime, err = timeFromID(start)
			if err != nil {
				err = fmt.Errorf("time from start query: %s", err)
				return
			}
		} else {
			// if the time is valid, update the start value to reflect keyTimeLayout
			start = startTime.Format(keyTimeLayout)
		}
	}

	// initialize page
	page = packages.Changelog{
		Records: make([]packages.ChangelogRecord, 0, limit),
	}

	domainID := []byte(domainRef)

	if err = s.DB.View(func(tx *bolt.Tx) (err error) {
		id, _ := getDomainIDByFQDN(tx, domainID)
		if id != nil {
			domainID = id
		}
		return
	}); err != nil {
		return
	}

	// get boltdbpool connection
	conn, err := s.Changelog.GetConnection(startTime)
	switch err {
	case timed.ErrUnknownDB:
		// get the closest connection
		conn, err = s.Changelog.PrevConnection(startTime)
		switch err {
		case timed.ErrUnknownDB:
			// unable to get any database in the past
			err = packages.ChangelogRecordNotFound
			return
		case nil:
			// there is no error
			break
		default:
			// unexpected error
			err = fmt.Errorf("get prev conn: %s", err)
			return
		}
		start = ""
		break
	case nil:
		// there is no error
		break
	default:
		// unexpected error
		err = fmt.Errorf("get conn: %s", err)
		return
	}
	// initialize the counter to track page.Next id value
	// the counter must not be greater the `limit`
	nextCounter := 0
	if err = conn.DB.View(func(tx *bolt.Tx) error {
		defer conn.Close()

		bucket := tx.Bucket(bucketNameChangelogDomainID)
		if bucket == nil {
			return nil
		}
		bucket = bucket.Bucket(domainID)
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		var k, v []byte
		if start == "" {
			// no start parameter, so get the latest record
			k, v = c.Last()
		} else {
			// go to the start record
			k, v = c.Seek([]byte(start))
			if k == nil {
				// if the record at the exact time does not exist
				// find the previous
				k, v = c.Prev()
			}
			for {
				// make sure that the time of the start record is not after the start time
				t, err := timeFromID(string(k))
				if err != nil {
					break
				}
				if !t.After(startTime) {
					break
				}
				k, v = c.Prev()
			}
			// find the next (newer) id for the next page
			var next []byte
			for ; nextCounter < limit; nextCounter++ {
				n, _ := c.Next()
				if n == nil {
					break
				}
				next = n
			}
			page.Next = string(next)
			if k != nil {
				// set the cursor back to the start key
				k, v = c.Seek(k)
			}
		}
		// get the records
		for i := 0; k != nil && i < limit; i++ {
			r := changelogRecord{}
			if err = json.Unmarshal(v, &r); err != nil {
				return err
			}
			r.id = string(k)
			r.domainID = string(domainID)
			record, err := r.export()
			if err != nil {
				return err
			}
			page.Records = append(page.Records, *record)
			k, v = c.Prev()
		}
		// set the previous (older) id for the previous record
		page.Previous = string(k)
		return nil
	}); err != nil {
		err = fmt.Errorf("next id update: %s", err)
		return
	}

	// update records from the previous database if needed
recordsLoop:
	for len(page.Records) < limit {
		conn, err = conn.Prev()
		switch err {
		case timed.ErrUnknownDB:
			// give up if there is no previous database
			err = nil
			break recordsLoop
		case nil:
		default:
			err = fmt.Errorf("get prev conn: %s", err)
			return
		}
		if err = conn.DB.View(func(tx *bolt.Tx) error {
			defer conn.Close()

			bucket := tx.Bucket(bucketNameChangelogDomainID)
			if bucket == nil {
				return nil
			}
			bucket = bucket.Bucket(domainID)
			if bucket == nil {
				return nil
			}
			c := bucket.Cursor()
			k, v := c.Last()
			for len(page.Records) < limit {
				r := changelogRecord{}
				if err = json.Unmarshal(v, &r); err != nil {
					return err
				}
				r.id = string(k)
				r.domainID = string(domainID)
				record, err := r.export()
				if err != nil {
					return err
				}
				page.Records = append(page.Records, *record)
				k, v = c.Prev()
				if k == nil {
					break
				}
			}
			page.Previous = string(k)
			return nil
		}); err != nil {
			err = fmt.Errorf("records update: %s", err)
			return
		}
	}
	// set the record count in the response
	page.Count = len(page.Records)

	// update previous from the previous database if it is blank
prevLoop:
	for page.Previous == "" && conn != nil {
		conn, err = conn.Prev()
		switch err {
		case timed.ErrUnknownDB:
			// give up if there is no previous database
			err = nil
			break prevLoop
		case nil:
		default:
			err = fmt.Errorf("get prev conn: %s", err)
			return
		}
		if err = conn.DB.View(func(tx *bolt.Tx) error {
			defer conn.Close()

			bucket := tx.Bucket(bucketNameChangelogDomainID)
			if bucket == nil {
				return nil
			}
			bucket = bucket.Bucket(domainID)
			if bucket == nil {
				return nil
			}
			c := bucket.Cursor()
			k, _ := c.Last()
			if k != nil {
				page.Previous = string(k)
			}
			return nil
		}); err != nil {
			err = fmt.Errorf("previous id update: %s", err)
			return
		}
	}

	// get the start database if the next id is not for the whole next page
	if nextCounter < limit {
		conn, err = s.Changelog.GetConnection(startTime)
		switch err {
		case timed.ErrUnknownDB:
			// All fine, there is no next database.
			err = nil
			return
		case nil:
			break
		default:
			err = fmt.Errorf("get conn for next id update: %s", err)
			return
		}
	}
	// update the next id until the next page is full
nextLoop:
	for nextCounter < limit {
		conn, err = conn.Next()
		switch err {
		case timed.ErrUnknownDB:
			err = nil
			break nextLoop
		case nil:
		default:
			err = fmt.Errorf("get next conn: %s", err)
			return
		}
		if err = conn.DB.View(func(tx *bolt.Tx) error {
			defer conn.Close()

			bucket := tx.Bucket(bucketNameChangelogDomainID)
			if bucket == nil {
				return nil
			}
			bucket = bucket.Bucket(domainID)
			if bucket == nil {
				return nil
			}
			c := bucket.Cursor()
			n, _ := c.First()
			var next []byte
			for ; nextCounter < limit; nextCounter++ {
				if n == nil {
					break
				}
				next = n
				n, _ = c.Next()
			}
			page.Next = string(next)
			return nil
		}); err != nil {
			err = fmt.Errorf("next id update: %s", err)
			return
		}
	}
	return
}
