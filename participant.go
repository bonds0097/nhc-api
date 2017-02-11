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

func GetParticipantsAdmin(w http.ResponseWriter, r *http.Request) {
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

	participants, errM := FindParticipants(db, user)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(participants)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func UpdateScorecard(w http.ResponseWriter, r *http.Request) {
	tokenData := GetToken(w, r)
	if tokenData == nil {
		return
	}

	type ScorecardData struct {
		ID        int     `bson:"id", json:"id"`
		Scorecard [][]int `bson:"scorecard", json:"scorecard"`
	}

	decoder := json.NewDecoder(r.Body)
	var scorecardData ScorecardData
	err := decoder.Decode(&scorecardData)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	user, errM := GetUserFromToken(db, tokenData)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	// Validate scorecard and update points.
	currentDay := time.Now().YearDay() - GLOBALS.ChallengeStart.YearDay()
	points := 0
	for i, week := range scorecardData.Scorecard {
		for j, day := range week {
			if i*7+j > currentDay {
				scorecardData.Scorecard[i][j], day = 0, 0
			} else if day > 0 {
				scorecardData.Scorecard[i][j], day = 1, 1
			} else {
				scorecardData.Scorecard[i][j], day = 0, 0
			}
			points += day
		}
	}
	user.Participants[scorecardData.ID].Points = points

	user.Participants[scorecardData.ID].Scorecard = scorecardData.Scorecard
	errM = user.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Scorecard updated successfully."}, http.StatusOK)
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

func FindParticipants(db *mgo.Database, u *User) (participants []Participant, errM *Error) {
	c := db.C("users")

	var users []User
	query := bson.M{"status": "registered"}

	// Org admins only see their own participants.
	if !(u.Role == GLOBAL_ADMIN.String() || u.Role == GLOBAL_SUPER_ADMIN.String()) {
		query["organization"] = u.Organization
	}

	// Get all the users.
	err := c.Find(query).All(&users)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving registered users: %s\n", err)), Internal: true}
		return
	}

	// Iterate through users and shove their participants in array.
	for _, user := range users {
		participants = append(participants, user.Participants...)
	}

	return
}
