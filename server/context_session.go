// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"net/http"
	"time"

	"gopherpit.com/gopherpit/services/session"
)

// saveSession saves session data using specified session service and
// sets a session cookie using http.ResponseWriter.
// The session is also cached in http.Request context.
func (s Server) saveSession(w http.ResponseWriter, r *http.Request, ses *session.Session, domain, path string) (rr *http.Request, err error) {
	defer func() {
		rr = r.WithContext(context.WithValue(r.Context(), contextSessionKey, ses))
	}()

	if ses == nil {
		ses = &session.Session{}
	}

	if ses.ID == "" {
		if ses, err = s.SessionService.CreateSession(&session.Options{
			Values: &ses.Values,
			MaxAge: &ses.MaxAge,
		}); err != nil {
			return
		}
	} else {
		if ses, err = s.SessionService.UpdateSession(ses.ID, &session.Options{
			Values: &ses.Values,
			MaxAge: &ses.MaxAge,
		}); err != nil {
			return
		}
	}

	if path == "" {
		path = "/"
	}
	cookie := &http.Cookie{
		Name:     s.SessionCookieName,
		Value:    ses.ID,
		Path:     path,
		Domain:   domain,
		MaxAge:   ses.MaxAge,
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	if cookie.MaxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	} else if cookie.MaxAge < 0 {
		cookie.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, cookie)
	return
}

func (s Server) session(r *http.Request) (ses *session.Session, rr *http.Request, err error) {
	rr = r
	if sv := r.Context().Value(contextSessionKey); sv != nil {
		var ok bool
		if ses, ok = sv.(*session.Session); ok {
			return
		}
	}
	defer func() {
		if ses != nil {
			rr = r.WithContext(context.WithValue(r.Context(), contextSessionKey, ses))
		}
	}()

	cookie, _ := r.Cookie(s.SessionCookieName)
	if cookie == nil {
		return
	}

	ses, err = s.SessionService.Session(cookie.Value)
	if err == session.SessionNotFound {
		ses, err = nil, nil
	}
	return
}

func (s Server) deleteSession(w http.ResponseWriter, r *http.Request, ses *session.Session) (*http.Request, error) {
	ses.MaxAge = -1
	return s.saveSession(w, r, ses, "", "")
}
