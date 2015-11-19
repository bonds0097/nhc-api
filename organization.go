package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Organization struct {
	ID   bson.ObjectId `bson:"_id" json:"-"`
	Name string        `bson:"name" json:"name"`
}

func GetOrganizations(w http.ResponseWriter, r *http.Request) {
	db := GetDB(w, r)
	organizations, errM := FindOrganizations(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, err := json.Marshal(organizations)
	if err != nil {
		ISR(w, r, errors.New(fmt.Sprintf("Failed to marshal organizations data: %s", err)))
	}
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func FindOrganizations(db *mgo.Database) (organizations []Organization, error *Error) {
	c := db.C("organizations")
	err := c.Find(nil).All(&organizations)
	if err != nil {
		error.Reason = errors.New(fmt.Sprintf("Error retrieving organizations from DB: %s", err))
		error.Internal = true
		return nil, error
	}

	return
}
