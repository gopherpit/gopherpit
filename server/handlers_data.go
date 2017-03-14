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

func dataDumpHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	srv.logger.Info("data dump: started")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.Join([]string{start.UTC().Format("2006-01-02T15-04-05Z0700"), srv.Name, version()}, "_")+`.tar"`)
	w.Header().Set("Date", start.UTC().Format(http.TimeFormat))

	tw := tar.NewWriter(w)
	var length int64

	srv.logger.Info("data dump: dumping salt data")
	header := &tar.Header{
		Name: srv.Name + ".salt",
		Mode: 0640,
		Size: int64(len(srv.salt)),
	}
	if err := tw.WriteHeader(header); err != nil {
		srv.logger.Errorf("data dump: write salt file header in tar: %s", err)
		return
	}
	n, err := tw.Write(srv.salt)
	if err != nil {
		srv.logger.Errorf("data dump: write salt file body in tar: %s", err)
		return
	}
	srv.logger.Infof("data dump: read %d bytes of salt data", n)

	services := []struct {
		Name    string
		Service interface{}
	}{
		{
			Name:    "certificate",
			Service: srv.CertificateService,
		},
		{
			Name:    "key",
			Service: srv.KeyService,
		},
		{
			Name:    "notification",
			Service: srv.NotificationService,
		},
		{
			Name:    "packages",
			Service: srv.PackagesService,
		},
		{
			Name:    "session",
			Service: srv.SessionService,
		},
		{
			Name:    "user",
			Service: srv.UserService,
		},
	}

	for _, service := range services {
		if u, ok := service.Service.(dataDump.Interface); ok {
			srv.logger.Infof("data dump: dumping %s service data", service.Name)
			dump, err := u.DataDump(nil)
			if err != nil {
				srv.logger.Errorf("data dump: read dump file %s: %s", dump.Name, err)
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
					srv.logger.Errorf("data dump: write file header %s in tar: %s", dump.Name, err)
					return
				}

				n, err := io.Copy(tw, dump.Body)
				defer dump.Body.Close()
				if err != nil {
					srv.logger.Errorf("data dump: write file data %s in tar: %s", dump.Name, err)
					return
				}
				length += n
				srv.logger.Infof("data dump: read %d bytes of %s service data", n, service.Name)
			}
		} else {
			srv.logger.Infof("data dump: skipping %s service dump", service.Name)
		}
	}

	if err := tw.Close(); err != nil {
		srv.logger.Errorf("data dump: closing tar: %s", err)
	}

	srv.logger.Infof("data dump: wrote %d bytes in %s", length, time.Since(start))
}
