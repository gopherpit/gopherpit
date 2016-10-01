// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"resenje.org/jsonresponse"
)

// ReadRequestBody unmarshals JSON encoded HTTP request body into
// an arbitrary interface. In case of error, it writes appropriate
// JSON-encoded response to http.ResponseWriter, so the calling handler
// should not write new data if this function returns error.
func ReadRequestBody(w http.ResponseWriter, r *http.Request, v interface{}) error {
	defer r.Body.Close()

	if r.Header.Get("Content-Length") == "0" {
		jsonresponse.BadRequest(w, jsonresponse.MessageResponse{
			Message: "empty request body",
		})
		return errors.New("empty request body")
	}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		response := jsonresponse.MessageResponse{}
		switch e := err.(type) {
		case *json.SyntaxError:
			response.Message = fmt.Sprintf("%v (offset %d)", e, e.Offset)
		case *json.UnmarshalTypeError:
			response.Message = fmt.Sprintf("expected json %s value but got %s (offset %d)", e.Type, e.Value, e.Offset)
		default:
			response.Message = err.Error()
		}
		jsonresponse.BadRequest(w, response)
		return err
	}
	return nil
}

// GetIPs returns all possible IPs found in HTTP request.
func GetIPs(r *http.Request) string {
	ips := []string{IPFromRemoteAddr(r.RemoteAddr)}
	xfr := r.Header.Get("X-Forwarded-For")
	if xfr != "" {
		ips = append(ips, xfr)
	}
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		ips = append(ips, xri)
	}
	return strings.Join(ips, ", ")
}

// IPFromRemoteAddr returns an IP without port from request's remote address.
func IPFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// GetHost returns request's host perpended with protocol: protocol://host.
func GetHost(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS == nil {
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + r.Host
}
