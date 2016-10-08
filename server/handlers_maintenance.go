// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"resenje.org/jsonresponse"
)

func (s Server) maintenanceHandler(h http.Handler, body string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(s.StorageDir, s.MaintenanceFilename)); err == nil {
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, body)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s Server) htmlMaintenanceHandler(h http.Handler) http.Handler {
	m, err := renderToString(s.template(tidMaintenance), "", nil)
	if err != nil {
		s.logger.Errorf("htmlMaintenanceHandler TemplateMaintenance error: %s", err)
		m = "Maintenance"
	}
	return s.maintenanceHandler(h, m, "text/html; charset=utf-8")
}

func (s Server) textMaintenanceHandler(h http.Handler) http.Handler {
	return s.maintenanceHandler(h, `Maintenance`, "text/plain; charset=utf-8")
}

func (s Server) jsonMaintenanceHandler(h http.Handler) http.Handler {
	return s.maintenanceHandler(h, `{"message":"maintenance","code":503}`, "application/json; charset=utf-8")
}

type maintenanceStatus struct {
	Status string `json:"status"`
}

func (s Server) maintenanceStatusAPIHandler(w http.ResponseWriter, r *http.Request) {
	status := "on"
	if _, err := os.Stat(filepath.Join(s.StorageDir, s.MaintenanceFilename)); os.IsNotExist(err) {
		status = "off"
	}
	jsonresponse.OK(w, maintenanceStatus{
		Status: status,
	})
}

func (s Server) maintenanceOnAPIHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(filepath.Join(s.StorageDir, s.MaintenanceFilename)); err == nil {
		jsonresponse.OK(w, nil)
		return
	}
	f, err := os.Create(filepath.Join(s.StorageDir, s.MaintenanceFilename))
	if err != nil {
		s.logger.Errorf("maintenance on: %s", err)
		jsonServerError(w, err)
		return
	}
	f.Close()
	s.logger.Info("maintenance on")
	jsonresponse.Created(w, nil)
}

func (s Server) maintenanceOffAPIHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(filepath.Join(s.StorageDir, s.MaintenanceFilename)); os.IsNotExist(err) {
		jsonresponse.OK(w, nil)
		return
	}
	if err := os.Remove(filepath.Join(s.StorageDir, s.MaintenanceFilename)); err != nil {
		s.logger.Errorf("maintenance off: %s", err)
		jsonServerError(w, err)
		return
	}
	s.logger.Info("maintenance off")
	jsonresponse.OK(w, nil)
}
