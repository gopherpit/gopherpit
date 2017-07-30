// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httpUser provides a HTTP client to an external
// user service that can respond to HTTP requests defined here.
package httpUser // import "gopherpit.com/gopherpit/services/user/http"

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"resenje.org/web/client/api"

	"gopherpit.com/gopherpit/services/user"
)

// Client implements gopherpit.com/gopherpit/services/user.Service interface.
type Client struct {
	*apiClient.Client
}

// NewClient creates a new Client.
func NewClient(c *apiClient.Client) *Client {
	if c == nil {
		c = &apiClient.Client{}
	}
	c.ErrorRegistry = errorRegistry
	return &Client{Client: c}
}

// User retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users/{ref}.
func (c Client) User(ref string) (u *user.User, err error) {
	u = &user.User{}
	err = c.JSON("GET", "/users/"+ref, nil, nil, u)
	err = getServiceError(err)
	return
}

// UserByID retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users-by-id/{id}.
func (c Client) UserByID(id string) (u *user.User, err error) {
	u = &user.User{}
	err = c.JSON("GET", "/users-by-id/"+id, nil, nil, u)
	err = getServiceError(err)
	return
}

// UserByEmail retrieves an existing User instance by making a HTTP GET request
// to {Client.Endpoint}/users-by-email/{email}.
func (c Client) UserByEmail(email string) (u *user.User, err error) {
	u = &user.User{}
	err = c.JSON("GET", "/users-by-email/"+email, nil, nil, u)
	err = getServiceError(err)
	return
}

// UserByUsername retrieves an existing User instance by making a HTTP GET
// request to {Client.Endpoint}/users-by-username/{username}.
func (c Client) UserByUsername(username string) (u *user.User, err error) {
	u = &user.User{}
	err = c.JSON("GET", "/users-by-username/"+username, nil, nil, u)
	err = getServiceError(err)
	return
}

// CreateUser creates a new User instance by making a HTTP POST request
// to {Client.Endpoint}/users. Post body is a JSON-encoded user.Options
// instance.
func (c Client) CreateUser(o *user.Options) (u *user.User, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	u = &user.User{}
	err = c.JSON("POST", "/users", nil, bytes.NewReader(body), u)
	err = getServiceError(err)
	return
}

// UpdateUser changes the data of an existing User by making a HTTP POST
// request to {Client.Endpoint}/users/{ref}. Post body is a JSON-encoded
// user.Options instance.
func (c Client) UpdateUser(ref string, o *user.Options) (u *user.User, err error) {
	body, err := json.Marshal(o)
	if err != nil {
		return
	}
	u = &user.User{}
	err = c.JSON("POST", "/users/"+ref, nil, bytes.NewReader(body), u)
	err = getServiceError(err)
	return
}

// DeleteUser deletes an existing User by making a HTTP DELETE request
// to {Client.Endpoint}/users/{ref}.
func (c Client) DeleteUser(ref string) (u *user.User, err error) {
	u = &user.User{}
	err = c.JSON("DELETE", "/users/"+ref, nil, nil, u)
	err = getServiceError(err)
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
func (c Client) RegisterUser(o *user.Options, password string, emailValidationDeadline time.Time) (u *user.User, emailValidationToken string, err error) {
	body, err := json.Marshal(RegisterUserRequest{
		Options:                 o,
		Password:                password,
		EmailValidationDeadline: emailValidationDeadline,
	})
	if err != nil {
		return
	}
	var response *RegisterUserResponse
	err = c.JSON("POST", "/register", nil, bytes.NewReader(body), response)
	err = getServiceError(err)
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
func (c Client) SetPassword(ref string, password string) (err error) {
	body, err := json.Marshal(SetPasswordRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	err = c.JSON("POST", "/users/"+ref+"/password", nil, bytes.NewReader(body), nil)
	err = getServiceError(err)
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
func (c Client) RequestPasswordReset(ref string) (token string, err error) {
	var response *RequestPasswordResetResponse
	err = c.JSON("POST", "/users/"+ref+"/password-reset-request", nil, nil, response)
	err = getServiceError(err)
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
func (c Client) ResetPassword(token, password string) (err error) {
	body, err := json.Marshal(ResetPasswordRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	err = c.JSON("POST", "/users/password-reset/"+token, nil, bytes.NewReader(body), nil)
	err = getServiceError(err)
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
func (c Client) RequestEmailChange(ref, email string, validationDeadline time.Time) (token string, err error) {
	body, err := json.Marshal(RequestEmailChangeRequest{
		Email:              email,
		ValidationDeadline: validationDeadline,
	})
	if err != nil {
		return
	}
	var response *RequestEmailChangeResponse
	err = c.JSON("POST", "/users/"+ref+"/email-change-request", nil, bytes.NewReader(body), response)
	err = getServiceError(err)
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
func (c Client) ChangeEmail(ref, token string) (u *user.User, err error) {
	body, err := json.Marshal(ChangeEmailRequest{
		Token: token,
	})
	if err != nil {
		return
	}
	u = &user.User{}
	err = c.JSON("POST", "/users/"+ref+"/email-change", nil, bytes.NewReader(body), u)
	err = getServiceError(err)
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
func (c Client) EmailChangeToken(ref, email string) (token string, err error) {
	var response *EmailChangeTokenResponse
	err = c.JSON("GET", "/users/"+ref+"/email-change/"+email, nil, nil, response)
	err = getServiceError(err)
	token = response.Token
	return
}

// UsersByID retrieves a paginated list of User instances ordered by ID values
// by making a HTTP GET request to
// {Client.Endpoint}/users-by-id?start={stardID}&limit={limit}.
func (c Client) UsersByID(startID string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startID != "" {
		query.Set("start", startID)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	page = &user.UsersPage{}
	err = c.JSON("GET", "/users-by-id", query, nil, page)
	err = getServiceError(err)
	return
}

// UsersByEmail retrieves a paginated list of User instances ordered by Email
// values by making a HTTP GET request to
// {Client.Endpoint}/users-by-email?start={stardEmail}&limit={limit}.
func (c Client) UsersByEmail(startEmail string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startEmail != "" {
		query.Set("start", startEmail)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	page = &user.UsersPage{}
	err = c.JSON("GET", "/users-by-email", query, nil, page)
	err = getServiceError(err)
	return
}

// UsersByUsername retrieves a paginated list of User instances ordered by
// Username values by making a HTTP GET request to
// {Client.Endpoint}/users-by-username?start={stardUsername}&limit={limit}.
func (c Client) UsersByUsername(startUsername string, limit int) (page *user.UsersPage, err error) {
	query := url.Values{}
	if startUsername != "" {
		query.Set("start", startUsername)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	page = &user.UsersPage{}
	err = c.JSON("GET", "/users-by-username", query, nil, page)
	err = getServiceError(err)
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
func (c Client) Authenticate(ref, password string) (u *user.User, err error) {
	body, err := json.Marshal(AuthenticateRequest{
		Password: password,
	})
	if err != nil {
		return
	}
	var response *AuthenticateResponse
	err = c.JSON("POST", "/users/"+ref+"/authenticate", nil, bytes.NewReader(body), response)
	err = getServiceError(err)
	u = response.User
	return
}
