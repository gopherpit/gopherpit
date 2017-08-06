// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"resenje.org/jsonresponse"
	"resenje.org/marshal"
	"resenje.org/web"

	"gopherpit.com/gopherpit/services/packages"
	"gopherpit.com/gopherpit/services/session"
	"gopherpit.com/gopherpit/services/user"
)

type authLoginRequest struct {
	Username   string           `json:"username"`
	Password   string           `json:"password"`
	RememberMe marshal.Checkbox `json:"remember-me"`
}

func (s *Server) authLoginFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	request := authLoginRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("auth login fe api: request decode: %s", err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Username == "" {
		errors.AddFieldError("username", "Email or username is required.")
	}
	if request.Password == "" {
		errors.AddFieldError("password", "Password is required.")
	}
	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}
	response, err := s.UserService.Authenticate(request.Username, request.Password)
	if err == user.ErrUserNotFound {
		s.Logger.Debugf("auth login fe api: authenticate: unknown user: %s", request.Username)
		jsonresponse.Unauthorized(w, nil)
		return
	}
	if err == user.ErrUnauthorized {
		s.Logger.Debugf("auth login fe api: authenticate: unauthorized: %s", request.Username)
		jsonresponse.Unauthorized(w, nil)
		return
	}
	if err != nil {
		s.Logger.Errorf("auth login fe api: authenticate: %s", err)
		jsonServerError(w, err)
		return
	}
	userID := response.ID
	if userID != "" {
		ses := &session.Session{
			Values: map[string]interface{}{
				"user-id": userID,
			},
		}
		if request.RememberMe {
			ses.MaxAge = s.RememberMeDays * 24 * 60 * 60
		}
		r, err = s.saveSession(w, r, ses, "", "")
		if err != nil {
			s.Logger.Errorf("auth login fe api: save session: %s", err)
			jsonServerError(w, err)
			return
		}
		s.Logger.Infof("auth login fe api: success: %s %s", userID, request.Username)

		s.auditf(r, nil, "login", "%s: %s", userID, request.Username)

		jsonresponse.OK(w, nil)
		return
	}
	s.Logger.Debugf("auth login fe api: unauthorized user: %s", request.Username)

	jsonresponse.Unauthorized(w, nil)
}

func (s *Server) authLogoutFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := s.logout(w, r); err != nil {
		s.Logger.Errorf("auth logout fe api: %s", err)
		jsonServerError(w, err)
		return
	}

	u, r, err := s.getRequestUser(r)
	if err != nil {
		s.audit(r, nil, "logout", "unknown user")
	} else {
		s.auditf(r, nil, "logout", "%s: %s", u.ID, u.Email)
	}

	jsonresponse.OK(w, nil)
}

type passwordResetTokenRequest struct {
	Username string `json:"username"`
}

func (s *Server) passwordResetTokenFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	request := passwordResetTokenRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("password reset token fe api: request decode: %s", err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Username == "" {
		s.Logger.Warning("password reset token fe api: empty username")
		errors.AddFieldError("username", "Email or username is required.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	u, err := s.UserService.User(request.Username)
	if err == user.ErrUserNotFound {
		s.Logger.Debugf("password reset token fe api: user: unknown user: %s", request.Username)
		errors.AddFieldError("username", "User not found.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if err != nil {
		s.Logger.Errorf("password reset token fe api: user %s: %s", request.Username, err)
		jsonServerError(w, err)
		return
	}

	optedOut, err := s.NotificationService.IsEmailOptedOut(u.Email)
	if err != nil {
		s.Logger.Errorf("password reset token fe api: is email %s opted out: %s", u.Email, err)
		jsonServerError(w, err)
		return
	}
	if optedOut {
		s.Logger.Warningf("password reset token fe api: email %s opted out", u.Email)
		errors.AddFieldError("username", "User's e-mail is opted-out.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	token, err := s.UserService.RequestPasswordReset(u.ID)
	if err != nil {
		if err == user.ErrUserNotFound {
			s.Logger.Warningf("password reset token fe api: request password reset: user not found: %s", u.ID)
			errors.AddFieldError("username", "User's e-mail is opted-out.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		s.Logger.Errorf("password reset token fe api: request password reset: %s", err)
		jsonServerError(w, err)
		return
	}

	s.Logger.Debugf("password reset token fe api: %s for email %s", token, u.Email)

	go func() {
		defer s.RecoveryService.Recover()
		if err := s.sendEmailPasswordResetEmail(r, u.Email, token); err != nil {
			msg := fmt.Sprintf("password reset token fe api: send email for email %s (token %s): %s", u.Email, token, err)
			if err := s.EmailService.Notify("Error: password reset email send error", msg); err != nil {
				s.Logger.Errorf("password reset token fe api: unable to send alert email: %s", err)
			}
			s.Logger.Error(msg)
		}
	}()

	s.Logger.Infof("password reset token fe api: success %s", request.Username)

	s.audit(r, nil, "password reset token", request.Username)

	jsonresponse.OK(w, nil)
}

type passwordResetRequest struct {
	Token     string `json:"token"`
	Password  string `json:"password1"`
	Password2 string `json:"password2"`
}

func (s *Server) passwordResetFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	request := passwordResetRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("password reset token fe api: request decode: %s", err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if len(request.Password) == 0 {
		s.Logger.Warningf("password reset fe api: empty password %s", request.Token)
		errors.AddFieldError("password1", "Password is required.")
	} else if len(request.Password) < 8 {
		s.Logger.Warningf("password reset fe api: short password %s", request.Token)
		errors.AddFieldError("password1", "Password is too short.")
	} else if request.Password != request.Password2 {
		s.Logger.Warningf("password reset fe api: password confirmation invalid %s", request.Token)
		errors.AddFieldError("password2", "Password is not confirmed.")
	}

	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	err := s.UserService.ResetPassword(request.Token, request.Password)
	if err != nil {
		if err == user.ErrPasswordResetTokenExpired {
			s.Logger.Warningf("password reset fe api: user token %s: %s", request.Token, err)
			errors.AddError("Password reset token has exprired.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		if err == user.ErrPasswordResetTokenNotFound {
			s.Logger.Warningf("password reset fe api: user token %s: %s", request.Token, err)
			errors.AddError("Password reset token is invalid.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		if err == user.ErrPasswordUsed {
			s.Logger.Warningf("password reset fe api: user token %s: %s", request.Token, err)
			errors.AddFieldError("password1", "This password has been used in the recent past.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		s.Logger.Errorf("password reset fe api: user token %s: %s", request.Token, err)
		jsonServerError(w, err)
		return
	}
	s.Logger.Infof("password reset: success token %s", request.Token)

	s.audit(r, nil, "password reset", request.Token)

	jsonresponse.OK(w, nil)
}

type userRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

func (s *Server) userFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("user fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Name == "" {
		s.Logger.Warningf("user fe api: name empty %s %s", u.ID, u.Email)
		errors.AddFieldError("name", "Your name is required.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	_, err = s.UserService.UpdateUser(u.ID, &user.Options{
		Name:     &request.Name,
		Username: &request.Username,
	})
	if err != nil {
		if err == user.ErrUsernameExists {
			s.Logger.Warningf("user fe api: user ID %s: %s", u.ID, err)
			errors.AddFieldError("username", "This username is already taken.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		if err == user.ErrUsernameInvalid {
			s.Logger.Warningf("user fe api: user ID %s: %s", u.ID, err)
			errors.AddFieldError("username", "This username is invalid.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		if err == user.ErrUsernameMissing {
			s.Logger.Warningf("user fe api: user ID %s: %s", u.ID, err)
			errors.AddFieldError("username", "Username is required.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		s.Logger.Errorf("user fe api: user ID %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}
	s.Logger.Infof("user fe api: success %s %s", u.ID, u.Email)

	s.auditf(r, request, "update user", "%s: %s", u.ID, u.Email)

	jsonresponse.OK(w, nil)
}

type userEmailRequest struct {
	Email string `json:"email"`
}

func (s *Server) userEmailFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userEmailRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("user email fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Email == "" {
		s.Logger.Warningf("user email fe api: email empty %s %s", u.ID, u.Email)
		errors.AddFieldError("email", "E-mail address is required.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if request.Email == u.Email {
		s.Logger.Debugf("user email fe api: same email %s %s", u.ID, u.Email)
		errors.AddFieldError("email", "New e-mail address is the same as the current one.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	token, err := s.UserService.RequestEmailChange(u.ID, request.Email, time.Now().Add(60*24*time.Hour))
	if err != nil {
		if err == user.ErrEmailChangeEmailNotAvaliable {
			s.Logger.Debugf("user email fe api: request email change: email %s is not available %s", request.Email, u.ID)
			errors.AddFieldError("email", "This e-mail address is not available.")
			jsonresponse.BadRequest(w, errors)
			return
		}
		s.Logger.Errorf("user email fe api: request email change: %s for user %s: %s", request.Email, u.ID, err)
		jsonServerError(w, err)
		return
	}

	if err := s.sendEmailValidationEmail(r, request.Email, token); err != nil {
		s.Logger.Errorf("user email fe api: send email validation: %s", err)
		jsonServerError(w, err)
		return
	}

	s.Logger.Infof("user email change fe api: success %s %s", u.ID, u.Email)

	s.auditf(r, nil, "user email change request", "%s: %s (token %s)", u.ID, request.Email, token)

	jsonresponse.OK(w, nil)
}

type userNotificationsSettingsRequest struct {
	NotificationsEnabled marshal.Checkbox `json:"notifications-enabled"`
}

func (s *Server) userNotificationsSettingsFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userNotificationsSettingsRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("user notifications settings fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	disabled := !request.NotificationsEnabled.Bool()

	if _, err = s.UserService.UpdateUser(u.ID, &user.Options{
		NotificationsDisabled: &disabled,
	}); err != nil {
		s.Logger.Errorf("user notifications settings fe api: update user %s: %s", u.ID, err)
		jsonServerError(w, err)
		return
	}

	s.Logger.Infof("user notifications settings fe api: success %s %s", u.ID, u.Email)

	s.auditf(r, request, "user notifications settings change", "%s: %s", u.ID, u.Email)

	jsonresponse.OK(w, nil)
}

type userPasswordRequest struct {
	Password     string `json:"password"`
	NewPassword1 string `json:"new-password1"`
	NewPassword2 string `json:"new-password2"`
}

func (s *Server) userPasswordFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userPasswordRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("user password fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if request.Password == "" {
		s.Logger.Debugf("user password fe api: new password empty %s %s", u.ID, u.Email)
		errors.AddFieldError("password", "Current password is required.")
	}
	if request.NewPassword1 == "" {
		s.Logger.Debugf("user password fe api: new password empty %s %s", u.ID, u.Email)
		errors.AddFieldError("new-password1", "Password is required.")
	} else if len(request.NewPassword1) < 8 {
		s.Logger.Debugf("user password fe api: new password too short %s %s", u.ID, u.Email)
		errors.AddFieldError("new-password1", "New password is too short.")
	}
	if request.NewPassword1 == "" {
		s.Logger.Debugf("user password fe api: new password empty %s %s", u.ID, u.Email)
		errors.AddFieldError("new-password2", "Password confirmation is required.")
	} else if request.NewPassword1 != request.NewPassword2 {
		s.Logger.Debugf("user password fe api: new passwords mismatch %s %s", u.ID, u.Email)
		errors.AddFieldError("new-password2", "Your new password is not confirmed.")
	}
	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	_, err = s.UserService.Authenticate(u.Email, request.Password)
	if err == user.ErrUnauthorized {
		s.Logger.Debugf("user password fe api: invalid password %s %s", u.ID, u.Email)
		errors.AddFieldError("password", "Invalid current password.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if err != nil && err != user.ErrUserNotFound {
		s.Logger.Errorf("user password fe api: authenticate %s %s: %s", u.ID, u.Email, err)
		jsonServerError(w, err)
		return
	}

	if err := s.UserService.SetPassword(u.ID, request.NewPassword1); err != nil {
		s.Logger.Errorf("user password api user: %s", err)
		jsonServerError(w, err)
		return
	}

	s.Logger.Infof("user password fe api: success %s %s", u.ID, u.Email)

	s.auditf(r, nil, "user password change", "%s: %s", u.ID, u.Email)

	jsonresponse.OK(w, nil)
}

type userDeleteRequest struct {
	Password string `json:"password"`
}

func (s *Server) userDeleteFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	request := userDeleteRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("user delete fe api: request decode %s %s: %s", u.ID, u.Email, err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	if request.Password == "" {
		s.Logger.Debugf("user delete fe api: empty password %s %s", u.ID, u.Email)
		errors.AddFieldError("password", "Your current password is required.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	_, err = s.UserService.Authenticate(u.Email, request.Password)
	if err == user.ErrUnauthorized {
		s.Logger.Debugf("user delete fe api: invalid password %s %s", u.ID, u.Email)
		errors.AddFieldError("password", "Invalid password.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if err != nil && err != user.ErrUserNotFound {
		s.Logger.Errorf("user delete api user: authenticate %s %s: %s", u.ID, u.Email, err)
		jsonServerError(w, err)
		return
	}

	domains, err := s.PackagesService.DomainsByOwner(u.ID, "", 1)
	if err != nil && err != packages.ErrUserDoesNotExist {
		s.Logger.Errorf("user delete fe api: domains by owner: %s", err)
		jsonServerError(w, err)
		return
	}

	if len(domains.Domains) > 0 {
		errors.AddError("You must transfer ownership or delete all domains that you own before account deletion.")
		jsonresponse.BadRequest(w, errors)
		return
	}

	deletedUser, err := s.UserService.DeleteUser(u.ID)
	if err != nil {
		s.Logger.Errorf("user delete fe api: delete user %s %s: %s", u.ID, u.Email, err)
		jsonServerError(w, err)
		return
	}
	s.Logger.Infof("user delete fe api: success id %s, email %s", deletedUser.ID, deletedUser.Email)

	s.auditf(r, nil, "user delete", "%s: %s", deletedUser.ID, deletedUser.Email)

	if _, err := s.logout(w, r); err != nil {
		s.Logger.Errorf("user delete logout: %s", err)
		jsonServerError(w, err)
		return
	}
	jsonresponse.OK(w, nil)
}

func (s *Server) userSendEmailValidationEmailFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	u, r, err := s.getRequestUser(r)
	if err != nil {
		panic(err)
	}

	token, err := s.UserService.EmailChangeToken(u.ID, u.Email)
	if err != nil {
		msg := fmt.Sprintf("user send email validation fe api: email cahnge token %s email %s: %s", u.ID, u.Email, err)
		if err := s.EmailService.Notify("Error: user send email validation", msg); err != nil {
			s.Logger.Errorf("user send email validation unable to send alert email: %s", err)
		}
		s.Logger.Error(msg)
		jsonServerError(w, err)
		return
	}
	if err := s.sendEmailValidationEmail(r, u.Email, token); err != nil {
		s.Logger.Errorf("user send email validation email fe api: %s %s: %s", u.ID, u.Email, err)
		jsonServerError(w, err)
		return
	}
	s.Logger.Infof("user send email validation api: success %s %s", u.ID, u.Email)

	s.auditf(r, nil, "user send email validation", "%s: %s (token %s)", u.ID, u.Email, token)

	jsonresponse.OK(w, nil)
}

type registrationRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password1"`
	Password2 string `json:"password2"`
}

func (s *Server) registrationFEAPIHandler(w http.ResponseWriter, r *http.Request) {
	request := registrationRequest{}
	errors := web.FormErrors{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.Logger.Warningf("registration fe api: request decode: %s", err)
		errors.AddError("Invalid data.")
		jsonresponse.BadRequest(w, errors)
		return
	}
	if request.Email == "" {
		s.Logger.Warning("registration fe api:: email empty")
		errors.AddFieldError("email", "E-mail is required.")
	} else {
		emailParts := strings.Split(request.Email, "@")
		if len(emailParts) != 2 {
			s.Logger.Warning("registration fe api: invalid email %s", request.Email)
			errors.AddFieldError("email", "E-mail address is invalid.")
		} else if _, err := net.ResolveIPAddr("ip", emailParts[1]); err != nil {
			s.Logger.Warning("registration fe api: invalid email domain %s", request.Email)
			errors.AddFieldError("email", "E-mail address has invalid domain.")
		} else {
			_, err := s.UserService.UserByEmail(request.Email)
			switch err {
			case nil:
				errors.AddFieldError("email", "Account with this e-mail address exists.")
			case user.ErrUserNotFound:
			default:
				s.Logger.Errorf("registration fe api: get user by email: %s", err)
				jsonServerError(w, err)
				return
			}
		}
	}
	if request.Username != "" {
		_, err := s.UserService.UserByUsername(request.Username)
		switch err {
		case nil:
			errors.AddFieldError("username", "This username is taken.")
		case user.ErrUserNotFound:
		default:
			s.Logger.Errorf("registration fe api: get user by username: %s", err)
			jsonServerError(w, err)
			return
		}
	}
	if request.Name == "" {
		s.Logger.Warningf("registration fe api: name empty %s", request.Email)
		errors.AddFieldError("name", "Your name is required.")
	}
	if len(request.Password) == 0 {
		s.Logger.Warningf("registration fe api: empty password %s", request.Email)
		errors.AddFieldError("password1", "Password is required.")
	} else if len(request.Password) < 8 {
		s.Logger.Warningf("registration fe api: short password %s", request.Email)
		errors.AddFieldError("password1", "Password is too short.")
	} else if request.Password != request.Password2 {
		s.Logger.Warningf("registration fe api: password confirmation invalid %s", request.Email)
		errors.AddFieldError("password2", "Password is not confirmed.")
	}
	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	admin := false
	emailUnvalidated := true
	disabled := false
	u, emailValidationToken, err := s.UserService.RegisterUser(&user.Options{
		Email:            &request.Email,
		Username:         &request.Username,
		Name:             &request.Name,
		Admin:            &admin,
		EmailUnvalidated: &emailUnvalidated,
		Disabled:         &disabled,
	}, request.Password, time.Now().Add(30*24*time.Hour))
	switch err {
	case user.ErrUsernameExists:
		errors.AddFieldError("username", "This username is taken.")
	case user.ErrUsernameInvalid:
		errors.AddFieldError("username", "This username is not valid.")
	case user.ErrEmailExists:
		errors.AddFieldError("email", "Account with this e-mail address exists.")
	case user.ErrEmailInvalid:
		errors.AddFieldError("email", "E-mail address is not valid.")
	case nil:
	default:
		s.Logger.Errorf("registration fe api: create user: %s", err)
		jsonServerError(w, err)
		return
	}
	if errors.HasErrors() {
		jsonresponse.BadRequest(w, errors)
		return
	}

	s.Logger.Debugf("registration fe api: email change %s token %s", u.Email, emailValidationToken)

	r, err = s.saveSession(w, r, &session.Session{
		Values: map[string]interface{}{
			"user-id": u.ID,
		},
	}, "", "")
	if err != nil {
		s.Logger.Errorf("registration fe api: session save: %s", err)
		jsonServerError(w, err)
		return
	}

	go func() {
		defer s.RecoveryService.Recover()
		if err := s.sendEmailValidationEmail(r, request.Email, emailValidationToken); err != nil {
			msg := fmt.Sprintf("registration fe api: validation email send for user id %s (token %s): %s", u.ID, emailValidationToken, err)
			if err := s.EmailService.Notify("Error: registration validation email send", msg); err != nil {
				s.Logger.Errorf("registration fe api: validation email unable to send alert email: %s", err)
			}
			s.Logger.Error(msg)
		}
	}()

	s.Logger.Infof("registration fe api: success %s %s", u.ID, request.Email)

	s.auditf(r, u, "registration", "%s: %s", u.ID, request.Email)

	jsonresponse.OK(w, nil)
}
