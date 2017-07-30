// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net"
	"time"
)

// TCPKeepAliveListener sets TCP keep alive period.
type TCPKeepAliveListener struct {
	*net.TCPListener
}

// NewTCPKeepAliveListener creates TCPKeepAliveListener
// from net.TCPListener.
func NewTCPKeepAliveListener(listener *net.TCPListener) TCPKeepAliveListener {
	return TCPKeepAliveListener{TCPListener: listener}
}

// Accept accepts TCP connection and sets TCP keep alive period
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
