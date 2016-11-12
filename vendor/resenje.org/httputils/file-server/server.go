// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer // import "resenje.org/httputils/file-server"

import (
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

// Server is a HTTP Handler tha serves static files from local filesystem.
type Server struct {
	Options

	root string
	dir  string

	hashes map[string]string
	mu     *sync.RWMutex
}

// New initializes a new instance of Server.
func New(root, dir string, options *Options) *Server {
	if options == nil {
		options = &Options{}
	}
	return &Server{
		Options: *options,

		root: root,
		dir:  dir,

		hashes: map[string]string{},
		mu:     &sync.RWMutex{},
	}
}

// ServeHTTP writes static files content to HTTP response.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
		r.URL.Path = urlPath
	}
	p := path.Clean(urlPath)

	if s.root != "" {
		if p = strings.TrimPrefix(p, s.root); len(p) >= len(r.URL.Path) {
			s.httpError(w, r, errNotFound)
			return
		}
	}

	if s.IndexPage != "" && strings.HasSuffix(r.URL.Path, s.IndexPage) {
		redirect(w, r, "./")
		return
	}

	if (s.Hasher != nil && !s.NoHashQueryStrings) ||
		(s.Hasher != nil && s.NoHashQueryStrings && len(r.URL.RawQuery) == 0) {
		cPath := s.canonicalPath(p)
		h, err := s.hash(cPath)
		if err != errNotRegularFile { // continue as usual if it is not a regular file
			if err != nil {
				s.httpError(w, r, err)
				return
			}
			if hPath := s.hashedPath(cPath, h); hPath != p {
				redirect(w, r, path.Join(s.root, hPath))
				return
			}
			if s.RedirectTrailingSlash && urlPath[len(urlPath)-1] == '/' {
				redirect(w, r, path.Join(s.root, p))
				return
			}
			p = cPath
			r.URL.Path = path.Join(s.root, cPath)
		}
	}
	f, err := open(s.dir, p)
	if err != nil {
		s.httpError(w, r, err)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		s.httpError(w, r, err)
		return
	}

	if s.RedirectTrailingSlash {
		url := r.URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				redirect(w, r, url+"/")
				return
			}
		} else {
			if url[len(url)-1] == '/' {
				redirect(w, r, "../"+path.Base(url))
				return
			}
		}
	}

	if d.IsDir() {
		index := strings.TrimSuffix(p, "/") + s.IndexPage
		ff, err := open(s.dir, index)
		if err == nil {
			defer ff.Close()
			dd, err := ff.Stat()
			if err == nil {
				p = index
				d = dd
				f = ff
			}
		}
	}

	if d.IsDir() {
		s.httpError(w, r, errNotFound)
		return
	}

	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}

// HashedPath returns a URL path with hash injected into the filename.
func (s *Server) HashedPath(p string) (string, error) {
	if s.Hasher == nil {
		return path.Join(s.root, p), nil
	}
	h, err := s.hash(p)
	if err != nil {
		return "", err
	}
	return path.Join(s.root, s.hashedPath(p, h)), nil
}

func (s Server) httpError(w http.ResponseWriter, r *http.Request, err error) {
	if os.IsNotExist(err) || err == errNotFound {
		if s.NotFoundHandler != nil {
			s.NotFoundHandler.ServeHTTP(w, r)
			return
		}
		DefaultNotFoundHandler.ServeHTTP(w, r)
		return
	}
	if os.IsPermission(err) {
		if s.ForbiddenHandler != nil {
			s.ForbiddenHandler.ServeHTTP(w, r)
			return
		}
		DefaultForbiddenHandler.ServeHTTP(w, r)
		return
	}
	if s.InternalServerErrorHandler != nil {
		s.InternalServerErrorHandler.ServeHTTP(w, r)
		return
	}
	DefaultInternalServerErrorHandler.ServeHTTP(w, r)
}

func (s *Server) hash(p string) (h string, err error) {
	s.mu.RLock()
	h, ok := s.hashes[p]
	s.mu.RUnlock()
	if ok {
		return
	}

	f, err := open(s.dir, p)
	if err != nil {
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		return
	}
	if !d.Mode().IsRegular() {
		err = errNotRegularFile
		return
	}

	h, err = s.Hasher.Hash(f)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.hashes[p] = h
	s.mu.Unlock()
	return
}

func (s Server) hashedPath(p, h string) string {
	if h == "" {
		return p
	}

	d, f := path.Split(p)

	i := strings.LastIndex(f, ".")
	if i > 0 {
		return d + f[:i] + "." + h + f[i:]
	}

	return d + f + "." + h
}

func (s Server) canonicalPath(p string) string {
	d, f := path.Split(p)

	parts := strings.Split(f, ".")
	f = ""
	l := len(parts)
	index := 1
	if l > 2 && !(l == 3 && parts[0] == "") {
		index = 2
	}
	for i, part := range parts {
		if i == l-index && s.Hasher.IsHash(part) {
			continue
		}
		if i != 0 {
			f += "."
		}
		f += part
	}

	return d + f
}
