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

func respond(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	if err := renderToResponse(w, tmpl, "", http.StatusOK, data, "text/html; charset=utf-8"); err != nil {
		panic(fmt.Sprintf("respond: %s", err))
	}
}

func respondText(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	if err := renderToResponse(w, tmpl, "", http.StatusOK, data, "text/plain; charset=utf-8"); err != nil {
		panic(fmt.Sprintf("respond text: %s", err))
	}
}

func (s Server) respond(w http.ResponseWriter, t string, data interface{}) {
	respond(w, s.templates[t], data)
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

func (s *Server) respondError(w http.ResponseWriter, r *http.Request, c int) {
	var ctx map[string]interface{}
	u, r, err := s.user(r)
	if err != nil {
		s.logger.Errorf("get user: %s", err)
		if _, ok := err.(net.Error); ok {
			if err := renderToResponse(w, s.templates["ServiceUnavailable"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
				s.logger.Errorf("render service unavailable response: %s", err)
			}
			return
		}
		if err := renderToResponse(w, s.templates["InternalServerError"], "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
			s.logger.Errorf("render internal server error response: %s", err)
		}
		return
	}
	var tpl *template.Template
	if u != nil {
		tpl = s.templates[errorTemplates[c][1]]
		ctx = map[string]interface{}{
			"User": u,
		}
	} else {
		tpl = s.templates[errorTemplates[c][0]]
	}
	if err := renderToResponse(w, tpl, "", c, ctx, "text/html; charset=utf-8"); err != nil {
		s.logger.Errorf("render http code %v response: %s", s, err)
	}
}

func (s *Server) htmlBadRequestHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusBadRequest)
}

func (s *Server) htmlUnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusUnauthorized)
}

func (s *Server) htmlForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusForbidden)
}

func (s *Server) htmlNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusNotFound)
}

func (s *Server) htmlRequestEntityTooLargeHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusRequestEntityTooLarge)
}

func (s *Server) htmlInternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusInternalServerError)
}

func (s *Server) htmlServiceUnavailableHandler(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusServiceUnavailable)
}
