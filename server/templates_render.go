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

func (s *Server) respondBadRequest(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusBadRequest)
}

func (s *Server) respondUnauthorized(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusUnauthorized)
}

func (s *Server) respondForbidden(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusForbidden)
}

func (s *Server) respondNotFound(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusNotFound)
}

func (s *Server) respondRequestEntityTooLarge(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusRequestEntityTooLarge)
}

func (s *Server) respondInternalServerError(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusInternalServerError)
}

func (s *Server) respondServiceUnavailable(w http.ResponseWriter, r *http.Request) {
	s.respondError(w, r, http.StatusServiceUnavailable)
}

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

func (s Server) respond(w http.ResponseWriter, t tid, data interface{}) {
	respond(w, s.template(t), data)
}

var errorTemplates = map[int][]tid{
	http.StatusBadRequest:            {tidBadRequest, tidBadRequestPrivate},
	http.StatusUnauthorized:          {tidUnauthorized, tidUnauthorizedPrivate},
	http.StatusForbidden:             {tidForbidden, tidForbiddenPrivate},
	http.StatusNotFound:              {tidNotFound, tidNotFoundPrivate},
	http.StatusRequestEntityTooLarge: {tidRequestEntityTooLarge, tidRequestEntityTooLargePrivate},
	http.StatusInternalServerError:   {tidInternalServerError, tidInternalServerErrorPrivate},
	http.StatusServiceUnavailable:    {tidServiceUnavailable, tidServiceUnavailablePrivate},
}

func (s *Server) respondError(w http.ResponseWriter, r *http.Request, c int) {
	var ctx map[string]interface{}
	u, r, err := s.user(r)
	if err != nil {
		s.logger.Errorf("get user: %s", err)
		if _, ok := err.(net.Error); ok {
			if err := renderToResponse(w, s.template(tidServiceUnavailable), "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
				s.logger.Errorf("render service unavailable response: %s", err)
			}
			return
		}
		if err := renderToResponse(w, s.template(tidInternalServerError), "", http.StatusServiceUnavailable, nil, "text/html; charset=utf-8"); err != nil {
			s.logger.Errorf("render internal server error response: %s", err)
		}
		return
	}
	var tpl *template.Template
	if u != nil {
		tpl = s.template(errorTemplates[c][1])
		ctx = map[string]interface{}{
			"User": u,
		}
	} else {
		tpl = s.template(errorTemplates[c][0])
	}
	if err := renderToResponse(w, tpl, "", c, ctx, "text/html; charset=utf-8"); err != nil {
		s.logger.Errorf("render http code %v response: %s", s, err)
	}
}
