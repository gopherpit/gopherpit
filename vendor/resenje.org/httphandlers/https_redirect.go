package httphandlers // import "resenje.org/httphandlers"

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

type TLSListener struct {
	*net.TCPListener
	TLSConfig *tls.Config
}

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

func HTTPToHTTPSRedirectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Location", "https://"+r.Host+r.RequestURI)
	w.WriteHeader(http.StatusMovedPermanently)
}
