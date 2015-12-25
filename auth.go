package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	privateKey = "/var/private/nhc.rsa"
)

var (
	signKey []byte
)

func init() {
	var err error
	signKey, err = ioutil.ReadFile(privateKey)
	if err != nil {
		log.Fatalf("Error reading Private Key: %s\n", err)
	}
}

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
		BR(w, r, errors.New("Unable to parse request."), http.StatusBadRequest)
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

	SetToken(w, r, user)
}

func Verify(w http.ResponseWriter, r *http.Request) {
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

func SetToken(w http.ResponseWriter, r *http.Request, user *User) {
	t := jwt.New(jwt.GetSigningMethod("RS256"))
	t.Claims["ID"] = user.ID.Hex()
	t.Claims["iat"] = time.Now().Unix()
	t.Claims["exp"] = time.Now().Add(time.Minute * 60 * 24 * 14).Unix()
	tokenString, err := t.SignedString(signKey)
	if err != nil {
		ISR(w, r, err)
		return
	}
	ServeJSON(w, r, &Response{"token": tokenString}, http.StatusOK)
	return
}

func GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	if IsTokenSet(r) {
		type UserData struct {
			Email     string `json:"email"`
			FirstName string `json:"firstName,omitempty"`
			LastName  string `json:"lastName,omitempty"`
			Picture   string `json:"picture,omitempty"`
			Role      string `json:"role,omitempty"`
			Status    string `json:"status,omitempty"`
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
		BR(w, r, errors.New("Missing Token. Please log in to continue."), http.StatusUnauthorized)
		return
	}
}
