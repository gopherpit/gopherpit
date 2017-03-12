// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"archive/tar"
	"io"
	"net/http"
	"strings"
	"time"

	"gopherpit.com/gopherpit/pkg/data-dump"
)

func (s Server) dataDumpHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.logger.Info("data dump: started")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.Join([]string{start.UTC().Format("2006-01-02T15-04-05Z0700"), s.Name, s.Version()}, "_")+`.tar"`)
	w.Header().Set("Date", start.UTC().Format(http.TimeFormat))

	tw := tar.NewWriter(w)
	var length int64

	s.logger.Info("data dump: dumping salt data")
	header := &tar.Header{
		Name: s.Name + ".salt",
		Mode: 0640,
		Size: int64(len(s.salt)),
	}
	if err := tw.WriteHeader(header); err != nil {
		s.logger.Errorf("data dump: write salt file header in tar: %s", err)
		return
	}
	n, err := tw.Write(s.salt)
	if err != nil {
		s.logger.Errorf("data dump: write salt file body in tar: %s", err)
		return
	}
	s.logger.Infof("data dump: read %d bytes of salt data", n)

	services := []struct {
		Name    string
		Service interface{}
	}{
		{
			Name:    "certificate",
			Service: s.CertificateService,
		},
		{
			Name:    "key",
			Service: s.KeyService,
		},
		{
			Name:    "notification",
			Service: s.NotificationService,
		},
		{
			Name:    "packages",
			Service: s.PackagesService,
		},
		{
			Name:    "session",
			Service: s.SessionService,
		},
		{
			Name:    "user",
			Service: s.UserService,
		},
	}

	for _, service := range services {
		if u, ok := service.Service.(dataDump.Interface); ok {
			s.logger.Infof("data dump: dumping %s service data", service.Name)
			dump, err := u.DataDump(nil)
			if err != nil {
				s.logger.Errorf("data dump: read dump file %s: %s", dump.Name, err)
				return
			}
			if dump != nil {
				header := &tar.Header{
					Name: dump.Name,
					Mode: 0640,
					Size: dump.Length,
				}
				if dump.ModTime != nil {
					header.ModTime = *dump.ModTime
				}
				if err := tw.WriteHeader(header); err != nil {
					s.logger.Errorf("data dump: write file header %s in tar: %s", dump.Name, err)
					return
				}

				n, err := io.Copy(tw, dump.Body)
				defer dump.Body.Close()
				if err != nil {
					s.logger.Errorf("data dump: write file data %s in tar: %s", dump.Name, err)
					return
				}
				length += n
				s.logger.Infof("data dump: read %d bytes of %s service data", n, service.Name)
			}
		} else {
			s.logger.Infof("data dump: skipping %s service dump", service.Name)
		}
	}

	if err := tw.Close(); err != nil {
		s.logger.Errorf("data dump: closing tar: %s", err)
	}

	s.logger.Infof("data dump: wrote %d bytes in %s", length, time.Since(start))
}
