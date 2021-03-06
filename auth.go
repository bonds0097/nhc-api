package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func Login(w http.ResponseWriter, r *http.Request) {
	type UserData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	var userData UserData
	err := decoder.Decode(&userData)
	if err != nil {
		BR(w, r, errors.New("Failed to parse request data."), http.StatusBadRequest)
		return
	}

	if userData.Email == "" || userData.Password == "" {
		BR(w, r, errors.New("Missing credentials"), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)

	user, errM := AuthUser(db, userData.Email, userData.Password)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	SetToken(w, r, user)
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithField("method", "SignUp")

	type UserData struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	var userData UserData
	err := decoder.Decode(&userData)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	if userData.Email == "" || userData.Password == "" {
		BR(w, r, errors.New("Missing information"), http.StatusBadRequest)
		return
	}

	// Generate confirmation code.
	confirmationCode, errM := GenerateConfirmationCode()
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	db := GetDB(w, r)
	user := &User{FirstName: userData.FirstName, LastName: userData.LastName, Email: userData.Email,
		Password: userData.Password, Status: UNCONFIRMED.String(), Role: USER.String(),
		Code: confirmationCode}
	errM = CreateUser(db, user)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	// Send confirmation e-mail if all went well.
	go SendVerificationMail(user)

	ctx.WithField("user", user.Email).Info("User signed up but needs confirmation.")

	SetToken(w, r, user)
}

func Verify(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithField("method", "Verify")

	type Message struct {
		Code string `json:"code"`
	}

	decoder := json.NewDecoder(r.Body)
	var message Message
	err := decoder.Decode(&message)
	if err != nil {
		BR(w, r, errors.New("Unable to parse request."), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	user, errM := FindUserByCode(db, message.Code)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	errM = user.Verify(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ctx.WithField("user", user.Email).Info("User successfully verified.")

	if !IsTokenSet(r) {
		SetToken(w, r, user)
		return
	}

	ServeJSON(w, r, &Response{"status": "ok"}, http.StatusOK)
	return
}

func ResendVerify(w http.ResponseWriter, r *http.Request) {
	if IsTokenSet(r) {

		tokenData := GetToken(w, r)
		db := GetDB(w, r)

		user, errM := GetUserFromToken(db, tokenData)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		errM = SendVerificationMail(user)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		ServeJSON(w, r, &Response{
			"status": fmt.Sprintf("Your verification e-mail has been resent to %s.", user.Email)},
			http.StatusOK)
	} else {
		BR(w, r, errors.New("Missing Token. Please log in to continue."), http.StatusUnauthorized)
		return
	}
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	type Message struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	var message Message
	err := decoder.Decode(&message)
	if err != nil {
		BR(w, r, errors.New("Unable to parse request."), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	user, errM := FindUserByEmail(db, message.Email)
	if errM != nil {
		ServeJSON(w, r, &Response{"status": "ok"}, http.StatusOK)
		return
	}

	// Generate code and save in DB. Then send email to user.
	code, _ := GenerateConfirmationCode()
	user.ResetCode = code
	errM = user.Save(db)
	if errM != nil {
		ServeJSON(w, r, &Response{"status": "ok"}, http.StatusOK)
		return
	}

	go SendResetPasswordMail(user)

	ServeJSON(w, r, &Response{"status": "ok"}, http.StatusOK)
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithField("method", "ResetPassword")

	type Message struct {
		Code            string `json:"code"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	decoder := json.NewDecoder(r.Body)
	var message Message
	err := decoder.Decode(&message)
	if err != nil {
		BR(w, r, errors.New("Unable to parse request."), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	user, errM := FindUserByResetCode(db, message.Code)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	// Make sure passwords match.
	if message.NewPassword != message.ConfirmPassword {
		BR(w, r, errors.New("Passwords do not match."), http.StatusBadRequest)
		return
	}

	// Update user.
	errM = ChangePassword(db, user, message.NewPassword)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ctx.WithField("user", user.Email).Info("User successfully changed password.")

	if !IsTokenSet(r) {
		SetToken(w, r, user)
		return
	}

	ServeJSON(w, r, &Response{"status": "ok"}, http.StatusOK)
	return
}

func SetToken(w http.ResponseWriter, r *http.Request, user *User) {
	ctx := logger.WithField("method", "SetToken")

	t := jwt.New(jwt.GetSigningMethod("RS256"))
	t.Claims["ID"] = user.ID.Hex()
	t.Claims["iat"] = time.Now().Unix()
	t.Claims["exp"] = time.Now().Add(time.Minute * 60 * 24 * 14).Unix()
	tokenString, err := t.SignedString(signKey)
	if err != nil {
		ISR(w, r, err)
		return
	}

	db := GetDB(w, r)
	user.LastLogin = time.Now()
	user.Save(db)
	ctx.WithField("user", user.Email).Debug("User token set.")

	ServeJSON(w, r, &Response{"token": tokenString}, http.StatusOK)
	return
}

func GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	if IsTokenSet(r) {
		type UserData struct {
			Email        string `json:"email"`
			FirstName    string `json:"firstName,omitempty"`
			LastName     string `json:"lastName,omitempty"`
			Organization string `json:"organization,omitempty"`
			Picture      string `json:"picture,omitempty"`
			Role         string `json:"role,omitempty"`
			Status       string `json:"status,omitempty"`
		}

		tokenData := GetToken(w, r)
		db := GetDB(w, r)

		user, errM := GetUserFromToken(db, tokenData)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		b, _ := json.Marshal(user)
		limitedUser := &UserData{}
		json.Unmarshal(b, limitedUser)
		b, _ = json.Marshal(limitedUser)
		parse := &Response{}
		json.Unmarshal(b, parse)
		ServeJSON(w, r, parse, http.StatusOK)
	} else {
		BR(w, r, errors.New(MISSING_TOKEN_ERROR), http.StatusUnauthorized)
		return
	}
}

func IsAuthorized(w http.ResponseWriter, r *http.Request, role string) bool {
	if IsTokenSet(r) {
		tokenData := GetToken(w, r)
		db := GetDB(w, r)

		user, errM := GetUserFromToken(db, tokenData)
		if errM != nil {
			HandleModelError(w, r, errM)
			return false
		}

		if strings.Contains(user.Role, role) {
			return true
		} else {
			BR(w, r, errors.New(FORBIDDEN_ERROR), http.StatusForbidden)
			return false
		}

	} else {
		BR(w, r, errors.New(MISSING_TOKEN_ERROR), http.StatusUnauthorized)
		return false
	}
}
