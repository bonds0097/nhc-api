package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/mgo.v2"
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

func FindGlobals(db *mgo.Database) (*Globals, error) {
	c := db.C("globals")
	var globals Globals
	err := c.Find(nil).One(&globals)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error retrieving globals from database: %s\n", err))
	}

	return &globals, nil
}
