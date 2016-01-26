package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Globals struct {
	ChallengeStart   time.Time `bson:"challengeStart,omitempty" json:"challengeStart,omitempty"`
	ChallengeEnd     time.Time `bson:"challengeEnd,omitempty" json:"challengeEnd,omitempty"`
	ChallengeLength  int       `bson:"challengeLength,omitempty" json:"challengeLength,omitempty"`
	RegistrationOpen bool      `bson:"registrationOpen,omitempty" json:"registrationOpen,omitempty"`
	ScorecardEnabled bool      `bson:"scorecardEnabled,omitempty" json:"scorecardEnabled,omitempty"`
}

func GetGlobals(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(GLOBALS)
	parse := &Response{}
	json.Unmarshal(b, parse)
	ServeJSON(w, r, parse, http.StatusOK)
}

func SaveGlobals(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var globals Globals
	err := decoder.Decode(&globals)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
	}
	globals.ChallengeLength = globals.ChallengeEnd.YearDay() - globals.ChallengeStart.YearDay() + 1

	db := GetDB(w, r)
	errM := UpdateGlobals(db, &globals)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	GLOBALS = &globals

	b, _ := json.Marshal(GLOBALS)
	parse := &Response{}
	json.Unmarshal(b, parse)
	ServeJSON(w, r, parse, http.StatusOK)
}

func UpdateGlobals(db *mgo.Database, globals *Globals) (errM *Error) {
	c := db.C("globals")
	err := c.Update(nil, bson.M{"$set": globals})
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error updating globals: %s\n", err))}
		return
	}

	return
}

func FindGlobals(db *mgo.Database) (*Globals, error) {
	c := db.C("globals")
	var globals Globals
	err := c.Find(nil).One(&globals)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error retrieving globals from database: %s\n", err))
	}

	return &globals, nil
}
