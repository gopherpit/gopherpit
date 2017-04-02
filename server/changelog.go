// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"time"

	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/user"
)

var (
	changelogActionDisplay = map[packages.Action]string{
		packages.ActionAddDomain:        "added domain",
		packages.ActionUpdateDomain:     "updated domain",
		packages.ActionDeleteDomain:     "deleted domain",
		packages.ActionDomainAddUser:    "added user to domain",
		packages.ActionDomainRemoveUser: "removed user from domain",
		packages.ActionAddPackage:       "added package",
		packages.ActionUpdatePackage:    "updated package",
		packages.ActionDeletePackage:    "deleted package",
	}
)

type changelogRecordChange struct {
	Field    string
	To       *string
	ToInfo   *string
	ToHref   *string
	From     *string
	FromInfo *string
	FromHref *string
}

type changelogAction packages.Action

func (c changelogAction) Display() string {
	return changelogActionDisplay[packages.Action(c)]
}

type changelogRecord struct {
	Time      time.Time
	DomainID  string
	FQDN      string
	PackageID string
	Path      string
	User      *user.User
	Action    changelogAction
	Changes   []changelogRecordChange
}

func (c changelogRecord) ImportPrefix() string {
	if c.FQDN == "" || c.Path == "" {
		return ""
	}
	return c.FQDN + c.Path
}

type changelog struct {
	Records  []changelogRecord
	Domain   packages.Domain
	Previous string
}

type changelogs []changelog

func (c changelogs) Len() int      { return len(c) }
func (c changelogs) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c changelogs) Less(i, j int) bool {
	if len(c[i].Records) == 0 {
		return false
	}
	if len(c[j].Records) == 0 {
		return true
	}
	return c[i].Records[0].Time.After(c[j].Records[0].Time)
}

func updateChangelogRecords(u user.User, record packages.ChangelogRecord, records *[]changelogRecord, users *map[string]*user.User) (err error) {
	if _, ok := (*users)[record.UserID]; !ok {
		ru, err := srv.UserService.UserByID(record.UserID)
		switch err {
		case nil:
			(*users)[record.UserID] = ru
		case user.ErrUserNotFound:
			(*users)[record.UserID] = nil
		default:
			return err
		}
	}

	r := changelogRecord{
		Time:      record.Time,
		DomainID:  record.DomainID,
		FQDN:      record.FQDN,
		PackageID: record.PackageID,
		Path:      record.Path,
		Action:    changelogAction(record.Action),
		Changes:   []changelogRecordChange{},
	}
	if ru, ok := (*users)[record.UserID]; ok {
		r.User = ru
	}
	if r.User == nil {
		r.User = &user.User{
			ID: record.UserID,
		}
	}

	var c changelogRecordChange
	switch record.Action {
	case packages.ActionAddDomain, packages.ActionUpdateDomain:
	Loop1:
		for _, change := range record.Changes {
			c = changelogRecordChange{}
			switch change.Field {
			case "owner-user-id":
				c.Field = "Owner"
				if change.To != nil {
					if _, ok := (*users)[*change.To]; !ok {
						ru, err := srv.UserService.UserByID(*change.To)
						switch err {
						case nil:
							(*users)[*change.To] = ru
						case user.ErrUserNotFound:
							(*users)[*change.To] = nil
						default:
							return err
						}
					}
					if *change.To == u.ID {
						c.To = stringToPtr("You")
					} else {
						if (*users)[*change.To] == nil {
							c.To = change.To
						} else {
							c.To = stringToPtr((*users)[*change.To].Name)
							c.ToHref = stringToPtr("/user/" + *change.To)
							c.ToInfo = stringToPtr("ID: " + *change.To)
						}
					}
				}

				if change.From != nil {
					if _, ok := (*users)[*change.From]; !ok {
						ru, err := srv.UserService.UserByID(*change.From)
						switch err {
						case nil:
							(*users)[*change.From] = ru
						case user.ErrUserNotFound:
							(*users)[*change.From] = nil
						default:
							return err
						}
					}
					if *change.From == u.ID {
						c.From = stringToPtr("You")
					} else {
						if (*users)[*change.From] == nil {
							c.From = change.From
						} else {
							c.From = stringToPtr((*users)[*change.From].Name)
							c.FromHref = stringToPtr("/user/" + *change.From)
							c.FromInfo = stringToPtr("ID: " + *change.From)
						}
					}
				}
			case "fqdn":
				c.Field = "FQDN"
				c.To = change.To
				c.From = change.From
			case "certificate-ignore":
				if record.Action == packages.ActionAddDomain {
					continue Loop1
				}
				c.Field = "Ignore TLS Certificate"
				c.To = change.To
				c.From = change.From
			case "disabled":
				c.Field = "Disabled"
				c.To = change.To
				c.From = change.From
			default:
				continue Loop1
			}
			r.Changes = append(r.Changes, c)
		}
	case packages.ActionDeleteDomain:
	case packages.ActionDomainAddUser, packages.ActionDomainRemoveUser:
	Loop2:
		for _, change := range record.Changes {
			c = changelogRecordChange{}
			switch change.Field {
			case "id":
				c.Field = "User"
				if change.To != nil {
					if _, ok := (*users)[*change.To]; !ok {
						ru, err := srv.UserService.UserByID(*change.To)
						switch err {
						case nil:
							(*users)[*change.To] = ru
						case user.ErrUserNotFound:
							(*users)[*change.To] = nil
						default:
							return err
						}
					}
					if (*users)[*change.To] == nil {
						c.To = change.To
					} else if *change.To == u.ID {
						c.To = stringToPtr("You")
					} else {
						c.To = stringToPtr((*users)[*change.To].Name)
						c.ToHref = stringToPtr("/user/" + *change.To)
						c.ToInfo = stringToPtr("ID: " + *change.To)
					}
				}

				if change.From != nil {
					if _, ok := (*users)[*change.From]; !ok {
						ru, err := srv.UserService.User(*change.From)
						if err != nil {
							if err != user.ErrUserNotFound {
								(*users)[*change.From] = nil
							}
							return err
						}
						(*users)[*change.From] = ru
					}
					if (*users)[*change.From] == nil {
						c.From = change.From
					} else if *change.From == u.ID {
						c.From = stringToPtr("You")
					} else {
						c.From = stringToPtr((*users)[*change.From].Name)
						c.FromHref = stringToPtr("/user/" + *change.From)
						c.FromInfo = stringToPtr("ID: " + *change.From)
					}
				}
			default:
				continue Loop2
			}
			r.Changes = append(r.Changes, c)
		}
	case packages.ActionAddPackage, packages.ActionUpdatePackage:
	Loop3:
		for _, change := range record.Changes {
			c = changelogRecordChange{}
			switch change.Field {
			case "domain-id", "domain-fqdn":
				continue Loop3
			case "path":
				c.Field = "Path"
				c.To = change.To
				c.From = change.From
			case "vcs":
				c.Field = "VCS"
				c.To = change.To
				c.From = change.From
			case "repo-root":
				c.Field = "Repository"
				c.To = change.To
				c.From = change.From
			case "ref-type":
				c.Field = "Reference type"
				c.To = change.To
				c.From = change.From
			case "ref-name":
				c.Field = "Reference name"
				c.To = change.To
				c.From = change.From
			case "go-source":
				c.Field = "Go Source"
				c.To = change.To
				c.From = change.From
			case "redirect-url":
				c.Field = "Redirect URL"
				c.To = change.To
				c.From = change.From
			case "disabled":
				c.Field = "Disabled"
				c.To = change.To
				c.From = change.From
			default:
				continue Loop3
			}
			r.Changes = append(r.Changes, c)
		}
	case packages.ActionDeletePackage:
	default:
		for _, change := range record.Changes {
			r.Changes = append(r.Changes, changelogRecordChange{
				Field: change.Field,
				From:  change.From,
				To:    change.To,
			})
		}
	}

	*records = append(*records, r)
	return
}

func stringToPtr(s string) *string {
	return &s
}
