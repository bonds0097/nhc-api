package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

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
	Team         string        `bson:"team,omitempty" json:"team,omitempty"`
	Sharing      string        `bson:"sharing,omitempty" json:"sharing,omitempty"`
	Comment      string        `bson:"comment,omitempty" json:"comment,omitempty"`
	Referral     string        `bson:"referral,omitempty" json:"referral,omitempty"`
	Donation     string        `bson:"donation,omitempty" json:"donation,omitempty"`
	Picture      string        `bson:"picture,omitempty" json:"picture,omitempty"`
	Facebook     string        `bson:"facebook,omitempty" json:"facebook,omitempty"`
	Google       string        `bson:"google,omitempty" json:"google,omitempty"`
	Role         string        `bson:"role,omitempty" json:"role,omitempty"`
	Status       string        `bson:"status,omitempty" json:"status,omitempty"`
	Participants []Participant `bson:"participants,omitempty" json:"participants,omitempty"`
	ResetCode    string        `bson:"resetCode,omitempty" json:"-"`
	Code         string        `bson:"code,omitempty" json:"-"`
	CreatedOn    time.Time     `bson:"createdOn,omitempty" json:"createdOn,omitempty"`
	LastLogin    time.Time     `bson:"lastLogin,omitempty" json:"lastLogin,omitempty"`
}

type LimitedUser struct {
	ID           bson.ObjectId `bson:"_id" json:"-"`
	Email        string        `bson:"email" json:"email"`
	FirstName    string        `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName     string        `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Family       string        `bson:"family,omitempty" json:"family,omitempty"`
	Organization string        `bson:"organization,omitempty" json:"organization,omitempty"`
	Team         string        `bson:"team,omitempty" json:"team,omitempty"`
	Comment      string        `bson:"comment,omitempty" json:"comment,omitempty"`
	Referral     string        `bson:"referral,omitempty" json:"referral,omitempty"`
	Role         string        `bson:"role,omitempty" json:"role,omitempty"`
	Status       string        `bson:"status,omitempty" json:"status,omitempty"`
	LastLogin    time.Time     `bson:"lastLogin,omitempty" json:"lastLogin,omitempty"`
}

type UserEditData struct {
	Email        string `bson:"email" json:"email"`
	FirstName    string `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName     string `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Family       string `bson:"family,omitempty" json:"family,omitempty"`
	Organization string `bson:"organization,omitempty" json:"organization,omitempty"`
	Team         string `bson:"team,omitempty" json:"team,omitempty"`
	Role         string `bson:"role,omitempty" json:"role,omitempty"`
	Status       string `bson:"status,omitempty" json:"status,omitempty"`
}

func UpdateSelf(w http.ResponseWriter, r *http.Request) {
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

	type UserUpdateData struct {
		FirstName    string `json:"firstName,omitempty"`
		LastName     string `json:"lastName,omitempty"`
		Organization string `json:"organization,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	var userUpdateData UserUpdateData
	err := decoder.Decode(&userUpdateData)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	if userUpdateData.FirstName != "" {
		user.FirstName = userUpdateData.FirstName
	}

	if userUpdateData.LastName != "" {
		user.LastName = userUpdateData.LastName
	}

	// Changing your organization resets your role to user.
	if userUpdateData.Organization != "" {
		user.Organization = userUpdateData.Organization
		user.Role = USER.String()
	}

	errM = user.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "User profile updated successfully."}, http.StatusOK)
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
	ctx := logger.WithField("method", "User_Save")

	uC := db.C("users")
	_, err := uC.UpsertId(u.ID, bson.M{"$set": u})
	if err != nil {
		if mgo.IsDup(err) {
			ctx.WithError(err).WithField("user", u.Email).Warn("Failed to create user. User already exists.")
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
	ctx := logger.WithField("method", "CreateUser")

	uC := db.C("users")
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return &Error{Reason: errors.New("Couldn't hash password."), Internal: true}
	}
	u.Password = string(pwHash)
	u.ID = bson.NewObjectId()
	u.CreatedOn = time.Now()
	u.LastLogin = time.Now()
	err = uC.Insert(u)
	if mgo.IsDup(err) {
		ctx.WithError(err).WithField("user", u.Email).Warn("Failed to create user. User already exists.")
		return &Error{Reason: errors.New("User already exists. Please log in instead."), Internal: false, Code: 409}
	}
	return nil
}

func AuthUser(db *mgo.Database, email, password string) (*User, *Error) {
	ctx := logger.WithField("method", "AuthUser")

	uC := db.C("users")
	user := &User{}
	err := uC.Find(bson.M{"email": email}).One(user)

	if err == mgo.ErrNotFound || user.ID == "" {
		ctx.WithError(err).WithField("email", email).Warn("User autentication failed because user does not exist.")
		return nil, &Error{Reason: errors.New("User wasn't found on our servers"), Internal: false}
	} else if err != nil {

		return nil, &Error{Reason: err, Internal: true}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		ctx.WithError(err).WithField("email", email).Warn("User authentication failed due to bad password.")
		return nil, &Error{Reason: errors.New("Incorrect password"), Internal: false, Code: http.StatusUnauthorized}
	}

	return user, nil
}

func FindUserByQuery(db *mgo.Database, query bson.M) (*User, *Error) {
	ctx := logger.WithField("method", "FindUserByQuery").WithField("query", query)

	uC := db.C("users")
	user := &User{}
	err := uC.Find(query).One(user)
	if err == mgo.ErrNotFound || user.ID == "" {
		ctx.WithError(err).Warn("User not found.")
		return nil, &Error{Reason: errors.New("No user found."), Internal: false, Code: http.StatusNotFound}
	} else if err != nil {
		ctx.WithError(err).Error("Failed to query for user.")
		return nil, &Error{Reason: err, Internal: true}
	}
	return user, nil
}

func FindUserById(db *mgo.Database, id bson.ObjectId) (*User, *Error) {
	ctx := logger.WithField("method", "FindUserById").WithField("id", id)

	uC := db.C("users")
	user := &User{}
	err := uC.FindId(id).One(user)

	if err == mgo.ErrNotFound || user.ID == "" {
		ctx.WithError(err).Warn("User not found.")
		return nil, &Error{Reason: errors.New("User not found."), Internal: false, Code: http.StatusUnauthorized}
	} else if err != nil {
		ctx.WithError(err).Error("Failed to query for user by id.")
		return nil, &Error{Reason: fmt.Errorf("mGo error: %s\n", err), Internal: true}
	}

	return user, nil
}

func FindUserByProvider(db *mgo.Database, provider, sub string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{provider: sub})
}

func FindUserByCode(db *mgo.Database, code string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{"code": code})
}

func FindUserByResetCode(db *mgo.Database, code string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{"resetCode": code})
}

func FindUserByEmail(db *mgo.Database, email string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{"email": email})
}

func ChangePassword(db *mgo.Database, u *User, password string) (errM *Error) {
	c := db.C("users")
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &Error{Reason: errors.New("Couldn't hash password."), Internal: true}
	}
	u.Password = string(pwHash)
	u.ResetCode = ""

	errM = u.Save(db)
	if errM != nil {
		return errM
	}

	update := bson.M{"$unset": bson.M{"resetCode": ""}}
	err = c.UpdateId(u.ID, update)
	if err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Failed to remove user's reset code: %s\n", err))}
		return
	}

	return
}

func UpdateUser(db *mgo.Database, u *UserEditData) *Error {
	c := db.C("users")
	err := c.Update(bson.M{"email": u.Email}, bson.M{"$set": u})
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Failed to update user: %s", err)), Internal: true}
	}

	return nil
}
