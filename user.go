package main

import (
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	ID        bson.ObjectId `bson:"_id" json:"-"`
	Email     string        `bson:"email" json:"email"`
	Password  string        `bson:"password,omitempty" json:"-"`
	FirstName string        `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName  string        `bson:"lastName,omitempty" json:"lastName,omitempty"`
	Picture   string        `bson:"picture,omitempty" json:"picture,omitempty"`
	Facebook  string        `bson:"facebook,omitempty" json:"facebook,omitempty"`
	Google    string        `bson:"google,omitempty" json:"google,omitempty"`
	Twitter   string        `bson:"twitter,omitempty" json:"twitter,omitempty"`
}

func (u *User) Save(db *mgo.Database) (err error) {
	uC := db.C("users")
	_, err = uC.UpsertId(u.ID, bson.M{"$set": u})
	return
}

func NewUser() (u *User) {
	u = &User{}
	u.ID = bson.NewObjectId()
	return
}

func CreateUser(db *mgo.Database, u *User) *Error {
	uC := db.C("users")
	pwHash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return &Error{Reason: errors.New("Couldn't hash password"), Internal: true}
	}
	u.Password = string(pwHash)
	u.ID = bson.NewObjectId()
	err = uC.Insert(u)
	if mgo.IsDup(err) {
		return &Error{Reason: errors.New("User already exists"), Internal: false}
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
	if err != nil {
		return nil, &Error{Reason: err, Internal: true}
	} else if user.ID == "" {
		return nil, &Error{Reason: errors.New("No user found"), Internal: false}
	}
	return user, nil
}

func FindUserById(db *mgo.Database, id bson.ObjectId) (*User, *Error) {
	uC := db.C("users")
	user := &User{}
	err := uC.FindId(id).One(user)
	if err != nil {
		return nil, &Error{Reason: err, Internal: true}
	} else if user.ID == "" {
		return nil, &Error{Reason: errors.New("No user found"), Internal: false}
	}
	return user, nil
}

func FindUserByProvider(db *mgo.Database, provider, sub string) (*User, *Error) {
	return FindUserByQuery(db, bson.M{provider: sub})
}
