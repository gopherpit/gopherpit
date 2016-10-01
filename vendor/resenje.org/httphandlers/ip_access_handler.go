package httphandlers // import "resenje.org/httphandlers"

import (
	"net"
	"net/http"
)

type IPAccessHandler struct {
	Handler             http.Handler
	UnauthorizedHandler http.Handler
	CIDRs               *[]string
	DenyAccess          bool
}

func (h IPAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.UnauthorizedHandler == nil {
		h.UnauthorizedHandler = http.HandlerFunc(defaultIPAccessForbiddenHandler)
	}

	if h.authenticate(r) == h.DenyAccess {
		h.forbidden(w, r)
		return
	}

	h.Handler.ServeHTTP(w, r)
}

func (h IPAccessHandler) authenticate(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(err)
	}
	ip := net.ParseIP(host)
	if len(*h.CIDRs) == 0 {
		return true
	}
	for _, cidr := range *h.CIDRs {
		_, cidrnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		if cidrnet.Contains(ip) {
			return true
		}
	}
	return false
}

func (h IPAccessHandler) forbidden(w http.ResponseWriter, r *http.Request) {
	h.UnauthorizedHandler.ServeHTTP(w, r)
}

func defaultIPAccessForbiddenHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}
