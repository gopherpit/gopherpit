// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package user // import "gopherpit.com/gopherpit/services/user"

import "time"

// User holds user account related data.
type User struct {
	ID                    string `json:"id"`
	Email                 string `json:"email"`
	Username              string `json:"username,omitempty"`
	Name                  string `json:"name,omitempty"`
	Admin                 bool   `json:"admin,omitempty"`
	NotificationsDisabled bool   `json:"notifications-disabled,omitempty"`
	EmailUnvalidated      bool   `json:"email-unvalidated,omitempty"`
	Disabled              bool   `json:"disabled,omitempty"`
}

func (u User) String() string {
	if u.Username != "" {
		return u.Username
	}
	return u.ID
}

// Options is a structure with parameters as pointers to set
// user data. If a parameter is nil, the corresponding User
// parameter will not be changed.
type Options struct {
	Email                 *string `json:"email,omitempty"`
	Username              *string `json:"username,omitempty"`
	Name                  *string `json:"name,omitempty"`
	Admin                 *bool   `json:"admin,omitempty"`
	NotificationsDisabled *bool   `json:"notifications-disabled,omitempty"`
	EmailUnvalidated      *bool   `json:"email-unvalidated,omitempty"`
	Disabled              *bool   `json:"disabled,omitempty"`
}

// UsersPage is a paginated list of User instances.
type UsersPage struct {
	Users []User `json:"users"`
	// Previous is an reference that
	// can be used to retrieve previous page.
	Previous string `json:"previous"`
	// Previous is an reference that
	// can be used to retrieve next page.
	Next string `json:"next"`
	// Count is a number of User instances in this UserPage.
	Count int `json:"count"`
}

// Service defines functions that User provider must have.
// Argument ref in some functions can be a string that is uniquely
// defined for a user: ID, Username or Email.
type Service interface {
	ManagementService
	RegisterService
	PasswordResetService
	EmailService
	Authenticator
}

// ManagementService defines most basic functionality for user management.
type ManagementService interface {
	// User retrieves a User instance.
	User(ref string) (*User, error)
	// UserByID retrieves a User instance only by it's ID.
	UserByID(id string) (*User, error)
	// UserByID retrieves a User instance only by it's Email.
	UserByEmail(email string) (*User, error)
	// UserByID retrieves a User instance only by it's Username.
	UserByUsername(username string) (*User, error)
	// Create user creates a new user interface.
	CreateUser(o *Options) (*User, error)
	// UpdateUser changes data of an existing User.
	UpdateUser(ref string, o *Options) (*User, error)
	// SetPassword changes a password of an existing User.
	SetPassword(ref string, password string) error
	// DeleteUser deletes an existing User.
	DeleteUser(ref string) (*User, error)

	// UsersByID retrieves a paginated list of User instances ordered by
	// ID values.
	UsersByID(startID string, limit int) (*UsersPage, error)
	// UsersByEmail retrieves a paginated list of User instances ordered by
	// Email values.
	UsersByEmail(startEmail string, limit int) (*UsersPage, error)
	// UsersByUsername retrieves a paginated list of User instances ordered by
	// Username values.
	UsersByUsername(startUsername string, limit int) (*UsersPage, error)
}

// RegisterService defines user registration interface.
type RegisterService interface {
	// RegisterUser is a method for adding new users.
	RegisterUser(o *Options, password string, emailValidationDeadline time.Time) (u *User, emailValidationToken string, err error)
}

// PasswordResetService handles password changes.
type PasswordResetService interface {
	// RequestPasswordReset starts a process of reseting a password by
	// providing a token that must be used in ResetPassword to authorize
	// password reset.
	RequestPasswordReset(ref string) (token string, err error)
	// ResetPassword changes a password of an existing User only if
	// provided token is valid.
	ResetPassword(token, password string) error
}

// EmailService handles e-mail changes.
type EmailService interface {
	// RequestEmailChange starts a process of changing an email by
	// returning a token that must be used in ChangeEmail to authorize
	// email change.
	RequestEmailChange(ref, email string, validationDeadline time.Time) (token string, err error)
	// ChangeEmail changes an email of an existing User only if
	// provided token is valid.
	ChangeEmail(ref, token string) (*User, error)
	// EmailChangeToken retrieves a token to change an email if it exists.
	EmailChangeToken(ref, email string) (token string, err error)
}

// Authenticator authenticates a User by a reference and password.
type Authenticator interface {
	// Authenticate validates a password of an existing User.
	Authenticate(ref, password string) (u *User, err error)
}

// Errors that are related to the User Services.
var (
	Unauthorized                 = NewError(401, "unauthorized")
	UserNotFound                 = NewError(1000, "user not found")
	SaltNotFound                 = NewError(1001, "salt not found")
	PasswordUsed                 = NewError(1002, "password already used")
	UsernameMissing              = NewError(1100, "username missing")
	UsernameInvalid              = NewError(1101, "username invalid")
	UsernameExists               = NewError(1102, "username exists")
	EmailMissing                 = NewError(1200, "email missing")
	EmailInvalid                 = NewError(1201, "email invalid")
	EmailExists                  = NewError(1202, "email exists")
	EmailChangeEmailNotAvaliable = NewError(1300, "email not available for change")
	EmailValidateTokenNotFound   = NewError(1301, "email validation token not found")
	EmailValidateTokenInvalid    = NewError(1302, "email validation token invalid")
	EmailValidateTokenExpired    = NewError(1303, "email validation token expired")
	PasswordResetTokenNotFound   = NewError(1400, "password reset token not found")
	PasswordResetTokenExpired    = NewError(1401, "password reset token expired")
)
