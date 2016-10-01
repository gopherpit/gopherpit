// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"net/http"

	"gopherpit.com/gopherpit/services/user"
)

// getUser retrieves a user.User from configured user service or from context.
func (s Server) user(r *http.Request) (u *user.User, rr *http.Request, err error) {
	rr = r
	if uv := r.Context().Value(contextUserKey); uv != nil {
		var ok bool
		if u, ok = uv.(*user.User); ok {
			return
		}
	}
	defer func() {
		if u != nil {
			rr = r.WithContext(context.WithValue(r.Context(), contextUserKey, u))
		}
	}()

	ses, rr, err := s.session(r)
	if err != nil || ses == nil {
		return
	}
	id, ok := ses.Values["user-id"].(string)
	if !ok || id == "" {
		return
	}

	u, err = s.UserService.UserByID(id)
	return
}

// logoutUser deletes session cookie and session data from session service.
func (s Server) logout(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	ses, _, err := s.session(r)
	if err != nil || ses == nil {
		return r, err
	}
	delete(ses.Values, "user-id")
	return s.deleteSession(w, r, ses)
}
