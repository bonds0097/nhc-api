package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Participant struct {
	ID               int     `bson:"id" json:"id"`
	FirstName        string  `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName         string  `bson:"lastName,omitempty" json:"lastName,omitempty"`
	AgeRange         [2]int  `bson:"ageRange,omitempty" json:"ageRange,omitempty"`
	Category         string  `bson:"category,omitempty" json:"category,omitempty"`
	Commitment       string  `bson:"commitment,omitempty" json:"commitment,omitempty"`
	CustomCommitment bool    `bson:"customCommitment,omitempty" json:"customCommitment,omitempty"`
	Scorecard        [][]int `bson:"scorecard,omitempty" json:"scorecard,omitempty"`
	Points           int     `bson:"points" json:"points"`
}

func GetParticipants(w http.ResponseWriter, r *http.Request) {
	if IsTokenSet(r) {
		tokenData := GetToken(w, r)
		db := GetDB(w, r)

		user, errM := GetUserFromToken(db, tokenData)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		b, err := json.Marshal(user.Participants)
		if err != nil {
			ISR(w, r, errors.New(fmt.Sprintf("Failed to marshal participant data: %s", err)))
		}
		ServeJSONArray(w, r, string(b), http.StatusOK)
	} else {
		BR(w, r, errors.New("Missing Token. Please log in to continue."), http.StatusUnauthorized)
		return
	}
}

func UpdateScorecard(w http.ResponseWriter, r *http.Request) {

}

func GenerateScorecard() (scorecard [][]int) {
	length := GLOBALS.ChallengeLength

	// Full weeks
	for i := 6; i < length; i += 7 {
		scorecard = append(scorecard, make([]int, 7, 7))
	}

	// Partial week
	scorecard = append(scorecard, make([]int, length%7, length%7))

	return
}
