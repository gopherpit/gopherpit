// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"net"
	"runtime"
	"strings"

	"github.com/miekg/dns"
)

func verifyDomain(domain, token string) (found bool, err error) {
	txt, _ := net.LookupTXT(domain)
	for _, r := range txt {
		if r == token {
			found = true
			return
		}
	}

	if runtime.GOOS == "windows" {
		return
	}

	resolv, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return
	}

	if len(resolv.Servers) == 0 {
		return
	}
	var index int
	d := domain
	c := &dns.Client{}
	m := &dns.Msg{}
	var in *dns.Msg
	for {
		index = strings.Index(d, ".")
		if index < 0 || index+1 >= len(d) {
			break
		}
		d = d[index+1:]

		m.SetQuestion(dns.Fqdn(d), dns.TypeNS)

		for _, ns := range resolv.Servers {
			in, _, err = c.Exchange(m, ns+":53")
			if err == nil {
				break
			}
		}
		if err != nil {
			return
		}

		for _, a := range in.Answer {
			ns, ok := a.(*dns.NS)
			if !ok {
				continue
			}
			m.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)
			in, _, err = c.Exchange(m, ns.Ns+":53")
			if err != nil {
				return
			}
			for _, a := range in.Answer {
				txt, ok := a.(*dns.TXT)
				if !ok {
					continue
				}
				for _, r := range txt.Txt {
					if r == token {
						return true, nil
					}
				}

			}
		}

		if len(in.Answer) > 0 {
			break
		}
	}
	return
}
