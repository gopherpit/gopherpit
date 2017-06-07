// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"html/template"
	"net"
	"net/http"
)

func renderToResponse(w http.ResponseWriter, tmpl *template.Template, name string, status int, data interface{}, contentType string) {
	if name == "" {
		name = "base"
	}
	buf := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		panic(err)
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		srv.Logger.Errorf("render to response: %v", err)
	}
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
	renderToResponse(w, tmpl, "", http.StatusOK, data, "text/plain; charset=utf-8")
}

func respond(w http.ResponseWriter, templateName string, data interface{}) {
	renderToResponse(w, srv.templates[templateName], "", http.StatusOK, data, "text/html; charset=utf-8")
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
		srv.Logger.Errorf("get user: %s", err)
		if _, ok := err.(net.Error); ok {
			renderToResponse(w, srv.templates["ServiceUnavailable"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8")
			return
		}
		renderToResponse(w, srv.templates["InternalServerError"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8")
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
	renderToResponse(w, tpl, "", c, ctx, "text/html; charset=utf-8")
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
