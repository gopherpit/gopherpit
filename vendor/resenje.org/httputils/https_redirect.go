// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"
)

type conn struct {
	net.Conn
	b byte
	e error
	f bool
}

func (c *conn) Read(b []byte) (int, error) {
	if c.f {
		c.f = false
		b[0] = c.b
		if len(b) > 1 && c.e == nil {
			n, e := c.Conn.Read(b[1:])
			if e != nil {
				c.Conn.Close()
			}
			return n + 1, e
		}
		return 1, c.e
	}
	return c.Conn.Read(b)
}

// TLSListener is a TCP listener that will check if the connection should be
// encrypted, and return encrypted connection, and if not, to return plain
// connection. It can be used along with HTTPToHTTPSRedirectHandler, to
// automatically redirect users from http:// to https:// protocol, by checking
// if http.Request has TLS equal to nil. Or to provide provide a different
// content in case that client provided or not TLS connection.
type TLSListener struct {
	*net.TCPListener
	TLSConfig *tls.Config
}

// Accept accepts TCP connection, sets keep alive and checks if a client
// requested an encrypted connection.
func (l TLSListener) Accept() (net.Conn, error) {
	c, err := l.AcceptTCP()
	if err != nil {
		return nil, err
	}
	c.SetKeepAlive(true)
	c.SetKeepAlivePeriod(3 * time.Minute)

	b := make([]byte, 1)
	_, err = c.Read(b)
	if err != nil {
		c.Close()
		if err != io.EOF {
			return nil, err
		}
	}

	con := &conn{
		Conn: c,
		b:    b[0],
		e:    err,
		f:    true,
	}

	if b[0] == 22 {
		return tls.Server(con, l.TLSConfig), nil
	}

	return con, nil
}

// HTTPToHTTPSRedirectHandler redirects with status code 301 to a https://
// version of HTTP request.
func HTTPToHTTPSRedirectHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL
	url.Scheme = "https"
	if url.Host == "" {
		url.Host = r.Host
	}
	w.Header().Set("Location", url.String())
	w.WriteHeader(http.StatusMovedPermanently)
}
