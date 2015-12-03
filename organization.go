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
	ID            bson.ObjectId `bson:"_id" json:"-"`
	Name          string        `bson:"name" json:"name"`
	NeedsApproval bool          `bson:"pending,omitempty" json:"-"`
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

func FindOrganizations(db *mgo.Database) (organizations []Organization, errM *Error) {
	c := db.C("organizations")
	err := c.Find(nil).All(&organizations)
	if err != nil {
		errM = &Error{
			Reason:   errors.New(fmt.Sprintf("Error retrieving organizations from DB: %s", err)),
			Internal: true}
		return
	}

	return
}

func CreateOrg(db *mgo.Database, org string) *Error {
	c := db.C("organizations")
	err := c.Insert(bson.M{"_id": bson.NewObjectId(), "name": org, "needsApproval": true})
	if err != nil && !mgo.IsDup(err) {
		return &Error{Reason: errors.New(fmt.Sprintf("Error creating new org: %s\n", err)), Internal: true}
	}

	return nil
}
