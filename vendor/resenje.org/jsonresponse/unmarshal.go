// Copyright (c) 2015, 2016 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonresponse

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// UnmarshalRequestBody unmarshals JSON encoded HTTP request body into
// an arbitrary interface. In case of error, it writes appropriate
// JSON-encoded response to http.ResponseWriter, so the calling handler
// should not write new data if this function returns error.
func UnmarshalRequestBody(w http.ResponseWriter, r *http.Request, v interface{}) error {
	defer r.Body.Close()

	if r.Header.Get("Content-Length") == "0" {
		BadRequest(w, MessageResponse{
			Message: "empty request body",
		})
		return errors.New("empty request body")
	}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		response := MessageResponse{}
		switch e := err.(type) {
		case *json.SyntaxError:
			response.Message = fmt.Sprintf("%v (offset %d)", e, e.Offset)
		case *json.UnmarshalTypeError:
			response.Message = fmt.Sprintf("expected json %s value but got %s (offset %d)", e.Type, e.Value, e.Offset)
		default:
			response.Message = err.Error()
		}
		BadRequest(w, response)
		return err
	}
	return nil
}
