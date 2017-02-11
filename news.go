package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type News struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	Subject     string        `bson:"subject" json:"subject"`
	Body        string        `bson:"body" json:"body"`
	Published   bool          `bson:"published" json:"published,omitempty"`
	PublishDate time.Time     `bson:"publishDate,omitempty" json:"publishDate,omitempty"`
	AdminOnly   bool          `bson:"adminOnly" json:"adminOnly,omitempty"`
}

func FetchNews(w http.ResponseWriter, r *http.Request) {
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

	getAdminNews := strings.Contains(user.Role, "admin")
	news, errM := FindPublishedNews(db, getAdminNews)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(news)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func ListNews(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	db := GetDB(w, r)
	news, errM := FindAllNews(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(news)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func AddNews(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var news News
	err := decoder.Decode(&news)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	// Make sure we have a subject and body and they're profanity free.
	if news.Subject == "" || news.Body == "" {
		BR(w, r, errors.New(MISSING_FIELDS_ERROR), http.StatusBadRequest)
		return
	}

	if HasProfanity(news.Subject) || HasProfanity(news.Body) {
		BR(w, r, errors.New(PROFANITY_ERROR), http.StatusBadRequest)
		return
	}

	// Otherwise, save the new item.
	if news.Published {
		news.PublishDate = time.Now()
	}

	news.ID = bson.NewObjectId()

	db := GetDB(w, r)
	errM := news.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Saved new item successfully."}, http.StatusOK)
	return
}

func DeleteNews(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	newsID := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	errM := RemoveNews(db, newsID)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "News item deleted."}, http.StatusOK)
}

func PublishNews(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	id := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	n, errM := FindNewsByID(db, id)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	n.Published = true
	n.PublishDate = time.Now()
	errM = n.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "News item published."}, http.StatusOK)
}

func UnpublishNews(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	id := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	n, errM := FindNewsByID(db, id)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	n.Published = false
	errM = n.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "News item unpublished."}, http.StatusOK)
}

func (n *News) Save(db *mgo.Database) (errM *Error) {
	c := db.C("news")
	_, err := c.UpsertId(n.ID, bson.M{"$set": n})
	if err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error saving news: %s\n", err))}
		return
	}

	return
}

func FindPublishedNews(db *mgo.Database, getAdminNews bool) (news []News, errM *Error) {
	query := bson.M{"published": true}
	if !getAdminNews {
		query["adminOnly"] = false
	}
	return FindNewsByQuery(db, query)
}

func FindAllNews(db *mgo.Database) (news []News, errM *Error) {
	return FindNewsByQuery(db, nil)
}

func FindNewsByQuery(db *mgo.Database, query bson.M) (news []News, errM *Error) {
	c := db.C("news")
	err := c.Find(query).All(&news)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving news: %s\n", err)), Internal: true}
		return
	}

	return
}

func FindNewsByID(db *mgo.Database, id bson.ObjectId) (news *News, errM *Error) {
	c := db.C("news")
	err := c.FindId(id).One(&news)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving news item: %s\n", err)), Internal: true}
		return
	}

	return
}

func RemoveNews(db *mgo.Database, id bson.ObjectId) (errM *Error) {
	c := db.C("news")
	err := c.RemoveId(id)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error removing news item: %s\n", err)), Internal: true}
		return
	}

	return
}
