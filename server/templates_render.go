// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"
)

func renderToResponse(w http.ResponseWriter, tmpl *template.Template, name string, status int, data interface{}, contentType string) (err error) {
	if name == "" {
		name = "base"
	}
	buf := bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	return
}

func renderToString(tmpl *template.Template, name string, data interface{}) (string, error) {
	if name == "" {
		name = "base"
	}
	buf := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func respondText(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	if err := renderToResponse(w, tmpl, "", http.StatusOK, data, "text/plain; charset=utf-8"); err != nil {
		panic(fmt.Sprintf("respond text: %s", err))
	}
}

func respond(w http.ResponseWriter, templateName string, data interface{}) {
	if err := renderToResponse(w, srv.templates[templateName], "", http.StatusOK, data, "text/html; charset=utf-8"); err != nil {
		panic(fmt.Sprintf("respond: %s", err))
	}
}

var errorTemplates = map[int][]string{
	http.StatusBadRequest:            {"BadRequest", "BadRequestPrivate"},
	http.StatusUnauthorized:          {"Unauthorized", "UnauthorizedPrivate"},
	http.StatusForbidden:             {"Forbidden", "ForbiddenPrivate"},
	http.StatusNotFound:              {"NotFound", "NotFoundPrivate"},
	http.StatusRequestEntityTooLarge: {"RequestEntityTooLarge", "RequestEntityTooLargePrivate"},
	http.StatusInternalServerError:   {"InternalServerError", "InternalServerErrorPrivate"},
	http.StatusServiceUnavailable:    {"ServiceUnavailable", "ServiceUnavailablePrivate"},
}

func respondError(w http.ResponseWriter, r *http.Request, c int) {
	var ctx map[string]interface{}
	u, r, err := getRequestUser(r)
	if err != nil {
		srv.logger.Errorf("get user: %s", err)
		if _, ok := err.(net.Error); ok {
			if err := renderToResponse(w, srv.templates["ServiceUnavailable"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
				srv.logger.Errorf("render service unavailable response: %s", err)
			}
			return
		}
		if err := renderToResponse(w, srv.templates["InternalServerError"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
			srv.logger.Errorf("render internal server error response: %s", err)
		}
		return
	}
	var tpl *template.Template
	if u != nil {
		tpl = srv.templates[errorTemplates[c][1]]
		ctx = map[string]interface{}{
			"User": u,
		}
	} else {
		tpl = srv.templates[errorTemplates[c][0]]
	}
	if err := renderToResponse(w, tpl, "", c, ctx, "text/html; charset=utf-8"); err != nil {
		srv.logger.Errorf("render http code %v response: %s", c, err)
	}
}

func htmlBadRequestHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusBadRequest)
}

func htmlUnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusUnauthorized)
}

func htmlForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusForbidden)
}

func htmlNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusNotFound)
}

func htmlRequestEntityTooLargeHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusRequestEntityTooLarge)
}

func htmlInternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusInternalServerError)
}

func htmlServiceUnavailableHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusServiceUnavailable)
}
