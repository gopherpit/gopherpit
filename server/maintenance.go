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

func maintenanceHandler(h http.Handler, body string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(srv.StorageDir, srv.MaintenanceFilename)); err == nil {
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

func htmlMaintenanceHandler(h http.Handler) http.Handler {
	m, err := renderToString(srv.templates["Maintenance"], "", nil)
	if err != nil {
		srv.Logger.Errorf("htmlMaintenanceHandler TemplateMaintenance error: %s", err)
		m = "Maintenance"
	}
	return maintenanceHandler(h, m, "text/html; charset=utf-8")
}

func textMaintenanceHandler(h http.Handler) http.Handler {
	return maintenanceHandler(h, `Maintenance`, "text/plain; charset=utf-8")
}

func jsonMaintenanceHandler(h http.Handler) http.Handler {
	return maintenanceHandler(h, `{"message":"Maintenance","code":503}`, "application/json; charset=utf-8")
}

type maintenanceStatus struct {
	Status string `json:"status"`
}

func maintenanceStatusAPIHandler(w http.ResponseWriter, r *http.Request) {
	status := "on"
	if _, err := os.Stat(filepath.Join(srv.StorageDir, srv.MaintenanceFilename)); os.IsNotExist(err) {
		status = "off"
	}
	jsonresponse.OK(w, maintenanceStatus{
		Status: status,
	})
}

func maintenanceOnAPIHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(filepath.Join(srv.StorageDir, srv.MaintenanceFilename)); err == nil {
		jsonresponse.OK(w, nil)
		return
	}
	f, err := os.Create(filepath.Join(srv.StorageDir, srv.MaintenanceFilename))
	if err != nil {
		srv.Logger.Errorf("maintenance on: %s", err)
		jsonServerError(w, err)
		return
	}
	f.Close()
	srv.Logger.Info("maintenance on")
	jsonresponse.Created(w, nil)
}

func maintenanceOffAPIHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(filepath.Join(srv.StorageDir, srv.MaintenanceFilename)); os.IsNotExist(err) {
		jsonresponse.OK(w, nil)
		return
	}
	if err := os.Remove(filepath.Join(srv.StorageDir, srv.MaintenanceFilename)); err != nil {
		srv.Logger.Errorf("maintenance off: %s", err)
		jsonServerError(w, err)
		return
	}
	srv.Logger.Info("maintenance off")
	jsonresponse.OK(w, nil)
}
