// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpUser provides a Service that is a HTTP client to an external
// user service that can respond to HTTP requests defined here.
package httpUser // import "gopherpit.com/gopherpit/services/user/http"

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"resenje.org/httputils/client/api"

	"gopherpit.com/gopherpit/services/user"
)

// Service implements gopherpit.com/gopherpit/services/user.Service interface.
type Service struct {
	Client *apiClient.Client
}

// NewService creates a new Service and injects user.ErrorRegistry
// in the API Client.
func NewService(c *apiClient.Client) *Service {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = user.ErrorRegistry
	return &Service{Client: c}
}

// User retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users/{ref}.
func (s Service) User(ref string) (u *user.User, err error) {
	err = s.Client.JSON("GET", "/users/"+ref, nil, nil, u)
	return
}

// UserByID retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users-by-id/{id}.
func (s Service) UserByID(id string) (u *user.User, err error) {
	err = s.Client.JSON("GET", "/users-by-id/"+id, nil, nil, u)
	return
}

// UserByEmail retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users-by-email/{email}.
func (s Service) UserByEmail(email string) (u *user.User, err error) {
	err = s.Client.JSON("GET", "/users-by-email/"+email, nil, nil, u)
	return
}

// UserByUsername retrieves an existing User instance by making a HTTP GET
// request to {Client.Endpoint}/users-by-username/{username}.
func (s Service) UserByUsername(username string) (u *user.User, err error) {
	err = s.Client.JSON("GET", "/users-by-username/"+username, nil, nil, u)
	return
}

// CreateUser creates a new User instance by making a HTTP POST request
// to {Client.Endpoint}/users. Post body is a JSON-encoded user.Options
// instance.
func (s Service) CreateUser(o *user.Options) (u *user.User, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/users", nil, bytes.NewReader(body), u)
	return
}

// UpdateUser changes the data of an existing User by making a HTTP POST
// request to {Client.Endpoint}/users/{ref}. Post body is a JSON-encoded
// user.Options instance.
func (s Service) UpdateUser(ref string, o *user.Options) (u *user.User, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/users/"+ref, nil, bytes.NewReader(body), u)
	return
}

// DeleteUser deletes an existing User by making a HTTP DELETE request
// to {Client.Endpoint}/users/{ref}.
func (s Service) DeleteUser(ref string) (u *user.User, err error) {
	err = s.Client.JSON("DELETE", "/users/"+ref, nil, nil, u)
	return
}

// RegisterUserRequest is a structure that is passed as JSON-encoded
// body to RegisterUser HTTP request.
type RegisterUserRequest struct {
	Options                 *user.Options `json:"options"`
	Password                string        `json:"password"`
	EmailValidationDeadline time.Time     `json:"email-validation-deadline"`
}

// RegisterUserResponse is expected structure of JSON-encoded response
// body for RegisterUser HTTP request.
type RegisterUserResponse struct {
	User                 *user.User `json:"user"`
	EmailValidationToken string     `json:"email-validation-token"`
}

// RegisterUser combines CreateUser SetPassword and RequestEmailChange
// into a single transaction to provide more convenient method
// for adding new users by making a HTTP POST request to
// {Client.Endpoint}/register. Request body is a
// JSON-encoded RegisterUserRequest instance. Expected response body
// is a JSON-encoded instance of RegisterUserResponse.
func (s Service) RegisterUser(o *user.Options, password string, emailValidationDeadline time.Time) (u *user.User, emailValidationToken string, err error) {
	body, err := json.Marshal(RegisterUserRequest{
		Options:                 o,
		Password:                password,
		EmailValidationDeadline: emailValidationDeadline,
	})
	if err != nil {
		return
	}
	var response *RegisterUserResponse
	err = s.Client.JSON("POST", "/register", nil, bytes.NewReader(body), response)
	u = response.User
	emailValidationToken = response.EmailValidationToken
	return
}

// SetPasswordRequest is a structure that is passed as JSON-encoded
// body to SetPassword HTTP request.
type SetPasswordRequest struct {
	Password string `json:"password"`
}

// SetPassword changes a password of an existing User by making a HTTP POST
// request to {Client.Endpoint}/users/{ref}/password. Request body is a
// JSON-encoded SetPasswordRequest instance.
func (s Service) SetPassword(ref string, password string) (err error) {
	body, err := json.Marshal(SetPasswordRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/users/"+ref+"/password", nil, bytes.NewReader(body), nil)
	return
}

// RequestPasswordResetResponse is expected structure of JSON-encoded response
// body for RequestPasswordReset HTTP request.
type RequestPasswordResetResponse struct {
	Token string `json:"token"`
}

// RequestPasswordReset starts a process of reseting a password by providing a
// token that must be used in ResetPassword to authorize password reset
// by making a HTTP POST request to
// {Client.Endpoint}/users/{ref}/password-reset-request. Expected response body
// is a JSON-encoded instance of RequestPasswordResetResponse.
func (s Service) RequestPasswordReset(ref string) (token string, err error) {
	var response *RequestPasswordResetResponse
	err = s.Client.JSON("POST", "/users/"+ref+"/password-reset-request", nil, nil, response)
	token = response.Token
	return
}

// ResetPasswordRequest is a structure that is passed as JSON-encoded body
// to ResetPassword HTTP request.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// ResetPassword changes a password of an existing User by making a HTTP POST
// request to {Client.Endpoint}/users/{ref}/password-reset. Request body is a
// JSON-encoded ResetPasswordRequest instance.
func (s Service) ResetPassword(token, password string) (err error) {
	body, err := json.Marshal(ResetPasswordRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/users/password-reset/"+token, nil, bytes.NewReader(body), nil)
	return
}

// RequestEmailChangeRequest is a structure that is passed as JSON-encoded body
// to RequestEmailChange HTTP request.
type RequestEmailChangeRequest struct {
	Email              string    `json:"email"`
	ValidationDeadline time.Time `json:"validation-deadline"`
}

// RequestEmailChangeResponse is expected structure of JSON-encoded response
// body for RequestEmailChange HTTP request.
type RequestEmailChangeResponse struct {
	Token string `json:"token"`
}

// RequestEmailChange starts a process of changing an email by returning
// a token that must be used in ChangeEmail to authorize email change by making
// a HTTP POST request to {Client.Endpoint}/users/{ref}/email-change-request.
// Request body is a JSON-encoded RequestEmailChangeRequest instance.
// Expected response body is a JSON-encoded instance of RequestEmailChangeResponse.
func (s Service) RequestEmailChange(ref, email string, validationDeadline time.Time) (token string, err error) {
	body, err := json.Marshal(RequestEmailChangeRequest{
		Email:              email,
		ValidationDeadline: validationDeadline,
	})
	if err != nil {
		return
	}
	var response *RequestEmailChangeResponse
	err = s.Client.JSON("POST", "/users/"+ref+"/email-change-request", nil, bytes.NewReader(body), response)
	token = response.Token
	return
}

// ChangeEmailRequest is a structure that is passed as JSON-encoded body
// to ChangeEmail HTTP request.
type ChangeEmailRequest struct {
	Token string `json:"token"`
}

// ChangeEmail changes an email of an existing User only if provided token is
// valid by making a HTTP POST request to
// {Client.Endpoint}/users/{ref}/email-change. Request body is a JSON-encoded
// ChangeEmailRequest instance.
func (s Service) ChangeEmail(ref, token string) (u *user.User, err error) {
	body, err := json.Marshal(ChangeEmailRequest{
		Token: token,
	})
	if err != nil {
		return
	}
	err = s.Client.JSON("POST", "/users/"+ref+"/email-change", nil, bytes.NewReader(body), u)
	return
}

// EmailChangeTokenResponse is expected structure of JSON-encoded response
// body for EmailChangeToken HTTP request.
type EmailChangeTokenResponse struct {
	Token string `json:"token"`
}

// EmailChangeToken retrieves a token to change an email, if it exists, by
// making a HTTP GET request to
// {Client.Endpoint}/users/{ref}/email-change/{email}.
func (s Service) EmailChangeToken(ref, email string) (token string, err error) {
	var response *EmailChangeTokenResponse
	err = s.Client.JSON("GET", "/users/"+ref+"/email-change/"+email, nil, nil, response)
	token = response.Token
	return
}

// UsersByID retrieves a paginated list of User instances ordered by ID values
// by making a HTTP GET request to
// {Client.Endpoint}/users-by-id?start={stardID}&limit={limit}.
func (s Service) UsersByID(startID string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startID != "" {
		query.Set("start", startID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/users-by-id", query, nil, page)
	return
}

// UsersByEmail retrieves a paginated list of User instances ordered by Email
// values by making a HTTP GET request to
// {Client.Endpoint}/users-by-email?start={stardEmail}&limit={limit}.
func (s Service) UsersByEmail(startEmail string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startEmail != "" {
		query.Set("start", startEmail)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/users-by-email", query, nil, page)
	return
}

// UsersByUsername retrieves a paginated list of User instances ordered by
// Username values by making a HTTP GET request to
// {Client.Endpoint}/users-by-username?start={stardUsername}&limit={limit}.
func (s Service) UsersByUsername(startUsername string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startUsername != "" {
		query.Set("start", startUsername)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	err = s.Client.JSON("GET", "/users-by-username", query, nil, page)
	return
}

// AuthenticateRequest is a structure that is passed as JSON-encoded body
// to Authenticate HTTP request.
type AuthenticateRequest struct {
	Password string `json:"password"`
}

// AuthenticateResponse is expected structure of JSON-encoded response
// body for Authenticate HTTP request.
type AuthenticateResponse struct {
	User *user.User `json:"user"`
}

// Authenticate validates a password of an existing User by making a HTTP POST
// request to {Client.Endpoint}/users/{ref}/authenticate. Request body is a
// JSON-encoded AuthenticateRequest instance. Expected response body is a
// JSON-encoded instance of AuthenticateResponse.
func (s Service) Authenticate(ref, password string) (u *user.User, err error) {
	body, err := json.Marshal(AuthenticateRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	var response *AuthenticateResponse
	err = s.Client.JSON("POST", "/users/"+ref+"/authenticate", nil, bytes.NewReader(body), response)
	u = response.User
	return
}
