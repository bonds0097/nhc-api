package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Commitment struct {
	ID          bson.ObjectId `bson:"_id" json:"-"`
	Name        string        `bson:"name" json:"name"`
	Urls        []string      `bson:"urls,omitempty" json:"urls,omitempty"`
	Commitments []string      `bson:"commitments,omitempty" json:"commitments,omitempty"`
}

func GetCommitments(w http.ResponseWriter, r *http.Request) {
	db := GetDB(w, r)
	commitments, errM := FindCommitments(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, err := json.Marshal(commitments)
	if err != nil {
		ISR(w, r, errors.New(fmt.Sprintf("Failed to marshal commitments data: %s", err)))
	}
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func FindCommitments(db *mgo.Database) (commitments []Commitment, error *Error) {
	c := db.C("commitments")
	err := c.Find(nil).All(&commitments)
	if err != nil {
		error.Reason = errors.New(fmt.Sprintf("Error retrieving commitments from DB: %s", err))
		error.Internal = true
		return nil, error
	}

	return
}
