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
	ID    bson.ObjectId `bson:"_id" json:"-"`
	Name  string        `bson:"name" json:"name"`
	Links []struct {
		Url   string `bson:"url,omitempty" json:"url,omitempty"`
		Title string `bson:"title,omitempty" json:"title,omitempty"`
	} `bson:"links,omitempty" json:"links,omitempty"`
	Commitments []string `bson:"commitments,omitempty" json:"commitments,omitempty"`
}

func GetCommitments(w http.ResponseWriter, r *http.Request) {
	db := GetDB(w, r)
	commitments, errM := FindCommitments(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(commitments)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func FindCommitments(db *mgo.Database) (commitments []Commitment, errM *Error) {
	c := db.C("commitments")
	err := c.Find(nil).All(&commitments)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving commitments from DB: %s", err)), Internal: true}
		return
	}

	return
}
