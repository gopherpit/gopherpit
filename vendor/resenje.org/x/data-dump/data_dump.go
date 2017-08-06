// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dataDump

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

// Interface defines method to retrieve data Dump. If ifModifiedSince
// is not nil and data is not changed since provided time,
// both return values, Dump and error, will be nil.
type Interface interface {
	DataDump(ifModifiedSince *time.Time) (dump *Dump, err error)
}

// Dump defines a structure that holds dump metadata and body as reader interface.
// Body must be closed after the read is done.
type Dump struct {
	Name        string
	ContentType string
	Length      int64
	ModTime     *time.Time
	Body        io.ReadCloser
}

// Logger defines methods required for logging.
type Logger interface {
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// stdLogger is a simple implementation of Logger interface
// that uses log package for logging messages.
type stdLogger struct{}

func (l stdLogger) Infof(format string, a ...interface{}) {
	log.Printf("INFO "+format, a...)
}

func (l stdLogger) Errorf(format string, a ...interface{}) {
	log.Printf("ERROR "+format, a...)
}

// Handler returns http.Handler that will call DataDump on
// every o field that implements Interface. If filePrefix is not blank
// Content-Disposition HTTP header will be added to the response.
// The response body will be the tar archive containing binary files
// named by the o fields that implement Interface.
func Handler(o interface{}, filePrefix string, logger Logger) http.Handler {
	if logger == nil {
		logger = stdLogger{}
	}
	if reflect.Indirect(reflect.ValueOf(o)).Kind() != reflect.Struct {
		panic(fmt.Sprintf("data dump: interface is not struct: %T", o))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Infof("data dump: started")

		w.Header().Set("Content-Type", "application/octet-stream")
		if filePrefix != "" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+strings.Join([]string{start.UTC().Format("2006-01-02T15-04-05Z0700"), filePrefix}, "_")+`.tar"`)
		}
		w.Header().Set("Date", start.UTC().Format(http.TimeFormat))

		tw := tar.NewWriter(w)
		var length int64

		v := reflect.Indirect(reflect.ValueOf(o))

		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			if u, ok := v.Field(i).Interface().(Interface); ok {
				name := v.Type().Field(i).Name
				logger.Infof("data dump: dumping %s data", name)
				dump, err := u.DataDump(nil)
				if err != nil {
					logger.Errorf("data dump: read dump file %s: %s", dump.Name, err)
					return
				}
				if dump != nil {
					header := &tar.Header{
						Name: dump.Name,
						Mode: 0666,
						Size: dump.Length,
					}
					if dump.ModTime != nil {
						header.ModTime = *dump.ModTime
					}
					if err := tw.WriteHeader(header); err != nil {
						logger.Errorf("data dump: write file header %s in tar: %s", dump.Name, err)
						return
					}

					n, err := io.Copy(tw, dump.Body)
					defer dump.Body.Close()
					if err != nil {
						logger.Errorf("data dump: write file data %s in tar: %s", dump.Name, err)
						return
					}
					length += n
					logger.Infof("data dump: read %d bytes of %s  data", n, name)
				}
			}
		}

		if err := tw.Close(); err != nil {
			logger.Errorf("data dump: closing tar: %s", err)
		}

		logger.Infof("data dump: wrote %d bytes in %s", length, time.Since(start))
	})
}
