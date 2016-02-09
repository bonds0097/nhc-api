package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Question struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	Text          string        `bson:"text" json:"text"`
	Answers       []string      `bson:"answers" json:"answers"`
	CorrectAnswer string        `bson:"correctAnswer" json:"correctAnswer,omitempty"`
	Enabled       bool          `bson:"enabled" json:"enabled,omitempty"`
	Respondents   []Respondent  `bson:"respondents" json:"respondents,omitempyty"`
}

type Respondent struct {
	User              string `bson:"user", json:"user"`
	AnsweredCorrectly bool   `bson:"answeredCorrectly,omitempty" json:"answeredCorrectly,omitempty"`
}

func FetchQuestion(w http.ResponseWriter, r *http.Request) {
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

	type UserQuestion struct {
		Enabled  bool `json:"enabled"`
		Question struct {
			Text    string   `json:"text"`
			Answers []string `json:"answers"`
		} `json:"question,omitempty"`
	}

	userQuestion := &UserQuestion{}

	question, errM := FindEnabledQuestion(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	if question == nil || question.AnsweredBy(user) {
		userQuestion.Enabled = false
	} else {
		userQuestion.Enabled = true
		b, _ := json.Marshal(question)
		json.Unmarshal(b, &userQuestion.Question)
	}

	b, _ := json.Marshal(userQuestion)
	parse := &Response{}
	json.Unmarshal(b, parse)
	ServeJSON(w, r, parse, http.StatusOK)
	return
}

func AnswerQuestion(w http.ResponseWriter, r *http.Request) {
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

	type AnswerData struct {
		Answer string `json:"answer"`
	}

	decoder := json.NewDecoder(r.Body)
	var data AnswerData
	err := decoder.Decode(&data)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	question, errM := FindEnabledQuestion(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	if question == nil || question.AnsweredBy(user) {
		BR(w, r, errors.New(FORBIDDEN_ERROR), http.StatusForbidden)
		return
	}

	var answeredCorrectly bool
	if question.CorrectAnswer == data.Answer {
		answeredCorrectly = true
	} else {
		answeredCorrectly = false
	}

	question.Respondents = append(question.Respondents, Respondent{User: user.Email,
		AnsweredCorrectly: answeredCorrectly})
	errM = question.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	var response string
	if answeredCorrectly {
		response = "Your submission was received and you answered the question correctly."
	} else {
		response = "Your submission was received but you answered the question incorrectly."
	}

	ServeJSON(w, r, &Response{"status": response}, http.StatusOK)
}

func GetQuestions(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	db := GetDB(w, r)
	questions, errM := FindAllQuestions(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(questions)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func CreateQuestion(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var question Question
	err := decoder.Decode(&question)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	// Validate question data
	if question.Text == "" || question.CorrectAnswer == "" || len(question.Answers) == 0 {
		BR(w, r, errors.New(MISSING_FIELDS_ERROR), http.StatusBadRequest)
		return
	}

	if HasProfanity(question.Text) || HasProfanity(question.CorrectAnswer) {
		BR(w, r, errors.New(PROFANITY_ERROR), http.StatusBadRequest)
		return
	}

	for _, answer := range question.Answers {
		if HasProfanity(answer) {
			BR(w, r, errors.New(PROFANITY_ERROR), http.StatusBadRequest)
			return
		}
	}

	// Save question
	db := GetDB(w, r)
	question.ID = bson.NewObjectId()
	errM := question.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Question successfully created."}, http.StatusOK)
}

func DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	id := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	errM := RemoveQuestion(db, id)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Question deleted."}, http.StatusOK)
}

func EnableQuestion(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	id := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	errM := DisableAllQuestions(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	errM = UpdateEnabledQuestion(db, id)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Question enabled."}, http.StatusOK)
}

func DisableQuestion(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	db := GetDB(w, r)
	errM := DisableAllQuestions(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Question disabled."}, http.StatusOK)
}

func (q *Question) Save(db *mgo.Database) *Error {
	c := db.C("questions")
	_, err := c.UpsertId(q.ID, bson.M{"$set": q})
	if err != nil {
		return &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error saving question: %s\n",
			err))}
	}

	return nil
}

func (q *Question) AnsweredBy(u *User) bool {
	for _, respondent := range q.Respondents {
		if u.Email == respondent.User {
			return true
		}
	}
	return false
}

func FindEnabledQuestion(db *mgo.Database) (q *Question, errM *Error) {
	c := db.C("questions")
	err := c.Find(bson.M{"enabled": true}).One(&q)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving enabled question: %s\n", err))}
		return
	}

	return
}

func FindAllQuestions(db *mgo.Database) (q []Question, errM *Error) {
	c := db.C("questions")
	err := c.Find(nil).All(&q)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving questions: %s\n", err))}
		return
	}

	return
}

func RemoveQuestion(db *mgo.Database, id bson.ObjectId) (errM *Error) {
	c := db.C("questions")
	err := c.RemoveId(id)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error removing question: %s\n", err)), Internal: true}
		return
	}

	return
}

func DisableAllQuestions(db *mgo.Database) *Error {
	questions, errM := FindAllQuestions(db)
	if errM != nil {
		return errM
	}

	for _, question := range questions {
		question.Enabled = false
		errM = question.Save(db)
		if errM != nil {
			return errM
		}
	}

	return nil
}

func UpdateEnabledQuestion(db *mgo.Database, id bson.ObjectId) *Error {
	c := db.C("questions")
	err := c.UpdateId(id, bson.M{"$set": bson.M{"enabled": true}})
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error enabling question: %s\n", err))}
	}

	return nil
}
