// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

// FormErrors represent structure errors returned by API server to
// request based on HTML form data.
type FormErrors struct {
	Errors      []string            `json:"errors,omitempty"`
	FieldErrors map[string][]string `json:"field-errors,omitempty"`
}

// AddError appends an error to a list of general errors.
func (f *FormErrors) AddError(e string) {
	f.Errors = append(f.Errors, e)
}

// AddFieldError appends an error to a list of field specific errors.
func (f *FormErrors) AddFieldError(field, e string) {
	if f.FieldErrors == nil {
		f.FieldErrors = map[string][]string{}
	}
	if _, ok := f.FieldErrors[field]; !ok {
		f.FieldErrors[field] = []string{}
	}
	f.FieldErrors[field] = append(f.FieldErrors[field], e)
}

// HasErrors returns weather FormErrors instance contains at leas one error.
func (f FormErrors) HasErrors() bool {
	if len(f.Errors) > 0 {
		return true
	}
	for _, v := range f.FieldErrors {
		if len(v) > 0 {
			return true
		}
	}
	return false
}

// NewError initializes FormErrors with one general error.
func NewError(e string) FormErrors {
	errors := FormErrors{}
	errors.AddError(e)
	return errors
}

// NewFieldError initializes FormErrors with one field error.
func NewFieldError(field, e string) FormErrors {
	errors := FormErrors{}
	errors.AddFieldError(field, e)
	return errors
}
