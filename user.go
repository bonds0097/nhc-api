package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	ID           bson.ObjectId `bson:"_id" json:"-"`
	Email        string        `bson:"email" json:"email"`
	Password     string        `bson:"password,omitempty" json:"-"`
	FirstName    string        `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName     string        `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Family       string        `bson:"family,omitempty" json:"family,omitempty"`
	Organization string        `bson:"organization,omitempty" json:"organization,omitempty"`
	Sharing      string        `bson:"sharing,omitempty" json:"sharing,omitempty"`
	Comment      string        `bson:"comment,omitempty" json:"comment,omitempty"`
	Donation     string        `bson:"donation,omitempty" json:"donation,omitempty"`
	Picture      string        `bson:"picture,omitempty" json:"picture,omitempty"`
	Facebook     string        `bson:"facebook,omitempty" json:"facebook,omitempty"`
	Google       string        `bson:"google,omitempty" json:"google,omitempty"`
	Role         string        `bson:"role,omitempty" json:"role,omitempty"`
	Status       string        `bson:"status,omitempty" json:"status,omitempty"`
	Participants []Participant `bson:"participants,omitempty" json:"participants,omitempty"`
	Code         string        `bson:"code,omitempty" json:"-"`
}

type LimitedUser struct {
	ID           bson.ObjectId `bson:"_id" json:"-"`
	Email        string        `bson:"email" json:"email"`
	FirstName    string        `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName     string        `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Family       string        `bson:"family,omitempty" json:"family,omitempty"`
	Organization string        `bson:"organization,omitempty" json:"organization,omitempty"`
	Comment      string        `bson:"comment,omitempty" json:"comment,omitempty"`
	Role         string        `bson:"role,omitempty" json:"role,omitempty"`
	Status       string        `bson:"status,omitempty" json:"status,omitempty"`
}

type UserEditData struct {
	Email        string `bson:"email" json:"email"`
	FirstName    string `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName     string `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Family       string `bson:"family,omitempty" json:"family,omitempty"`
	Organization string `bson:"organization,omitempty" json:"organization,omitempty"`
	Role         string `bson:"role,omitempty" json:"role,omitempty"`
	Status       string `bson:"status,omitempty" json:"status,omitempty"`
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "admin") {
		return
	}

	tokenData := GetToken(w, r)
	if tokenData == nil {
		return
	}

	db := GetDB(w, r)
	user, errM := GetUserFromToken(db, tokenData)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	users, errM := FindLimitedUsers(db, user)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(users)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func EditUser(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "admin") {
		return
	}

	tokenData := GetToken(w, r)
	if tokenData == nil {
		return
	}

	db := GetDB(w, r)
	callingUser, errM := GetUserFromToken(db, tokenData)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var userEditData UserEditData
	err := decoder.Decode(&userEditData)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	// Only global admins can change a user's role or status.
	if callingUser.Role != GLOBAL_ADMIN.String() && callingUser.Role != GLOBAL_SUPER_ADMIN.String() {
		userEditData.Role = ""
		userEditData.Status = ""
	}

	errM = UpdateUser(db, &userEditData)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "User successfully updated."}, http.StatusOK)
}

func (u *User) Save(db *mgo.Database) (errM *Error) {
	uC := db.C("users")
	_, err := uC.UpsertId(u.ID, bson.M{"$set": u})
	if err != nil {
		if mgo.IsDup(err) {
			errM = &Error{Internal: false, Reason: errors.New("That user already exists. Please login first."), Code: http.StatusConflict}
		} else {
			errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error updating user: %s\n", err))}
		}
		return
	}

	return
}

func (u *User) Verify(db *mgo.Database) (errM *Error) {
	uC := db.C("users")
	u.Status = UNREGISTERED.String()
	errM = u.Save(db)
	if errM != nil {
		return
	}

	update := bson.M{"$unset": bson.M{"code": ""}}
	err := uC.UpdateId(u.ID, update)
	if err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Failed to remove user's code: %s\n", err))}
		return
	}

	return
}

func NewUser() (u *User) {
	u = &User{}
	u.ID = bson.NewObjectId()
	return
}

func FindLimitedUsers(db *mgo.Database, u *User) (users []LimitedUser, errM *Error) {
	c := db.C("users")

	// If user is global admin, return all users. Otherwise just users in the user's org.
	if u.Role == GLOBAL_ADMIN.String() || u.Role == GLOBAL_SUPER_ADMIN.String() {
		err := c.Find(nil).All(&users)
		if err != nil {
			errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving users from DB: %s", err)), Internal: true}
			return
		}
	} else {
		err := c.Find(bson.M{"organization": u.Organization}).All(&users)
		if err != nil {
			errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving users from DB: %s", err)), Internal: true}
			return
		}
	}

	return
}

func CreateUser(db *mgo.Database, u *User) *Error {
	uC := db.C("users")
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return &Error{Reason: errors.New("Couldn't hash password."), Internal: true}
	}
	u.Password = string(pwHash)
	u.ID = bson.NewObjectId()
	err = uC.Insert(u)
	if mgo.IsDup(err) {
		return &Error{Reason: errors.New("User already exists. Please log in instead."), Internal: false, Code: 409}
	}
	return nil
}

func AuthUser(db *mgo.Database, email, password string) (*User, *Error) {
	uC := db.C("users")
	user := &User{}
	err := uC.Find(bson.M{"email": email}).One(user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, &Error{Reason: errors.New("User wasn't found on our servers"), Internal: false}
		}
		return nil, &Error{Reason: err, Internal: true}
	}
	if user.ID == "" {
		return nil, &Error{Reason: errors.New("No user found"), Internal: false}
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, &Error{Reason: errors.New("Incorrect password"), Internal: false, Code: http.StatusUnauthorized}
	}
	return user, nil

}

func FindUserByQuery(db *mgo.Database, query bson.M) (*User, *Error) {
	uC := db.C("users")
	user := &User{}
	err := uC.Find(query).One(user)
	if err == mgo.ErrNotFound {
		return nil, &Error{Reason: errors.New("No user found."), Internal: false, Code: http.StatusNotFound}
	} else if user.ID == "" {
		return nil, &Error{Reason: errors.New("No user found."), Internal: false, Code: http.StatusNotFound}
	} else if err != nil {
		return nil, &Error{Reason: err, Internal: true}
	}
	return user, nil
}

func FindUserById(db *mgo.Database, id bson.ObjectId) (*User, *Error) {
	uC := db.C("users")
	user := &User{}
	err := uC.FindId(id).One(user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, &Error{Reason: errors.New("User not found."), Internal: false, Code: http.StatusUnauthorized}
		} else {
			return nil, &Error{Reason: errors.New(fmt.Sprintf("mGo error: %s\n", err)), Internal: true}
		}
	} else if user.ID == "" {
		return nil, &Error{Reason: errors.New("No user found."), Internal: false, Code: http.StatusUnauthorized}
	}
	return user, nil
}

func FindUserByProvider(db *mgo.Database, provider, sub string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{provider: sub})
}

func FindUserByCode(db *mgo.Database, code string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{"code": code})
}

func UpdateUser(db *mgo.Database, u *UserEditData) *Error {
	c := db.C("users")
	err := c.Update(bson.M{"email": u.Email}, bson.M{"$set": u})
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Failed to update user: %s", err)), Internal: true}
	}

	return nil
}
