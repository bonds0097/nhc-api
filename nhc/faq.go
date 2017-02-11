package nhc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type FAQ struct {
	ID       bson.ObjectId `bson:"_id" json:"id"`
	Question string        `bson:"question" json:"question"`
	Answer   string        `bson:"answer" json:"answer"`
	Category string        `bson:"category" json:"category"`
}

// GetFaqs gets and returns all frequently asked questions
func GetFaqs(w http.ResponseWriter, r *http.Request) {
	db := GetDB(w, r)
	faqs, errM := FindAllFaqs(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, _ := json.Marshal(faqs)
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

// AddFaq /admin creates a new frequently asked question
func AddFaq(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var faq FAQ
	err := decoder.Decode(&faq)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	// Validate faq data
	if faq.Question == "" || faq.Answer == "" || faq.Category == "" {
		BR(w, r, errors.New(MISSING_FIELDS_ERROR), http.StatusBadRequest)
		return
	}

	// Create faq before saving?
	// Save faq
	db := GetDB(w, r)
	faq.ID = bson.NewObjectId()
	errM := faq.Save(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "FAQ successfully added."}, http.StatusOK)
}

// EditFaq /admin updates an existing frequently asked question
func EditFaq(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var faq FAQ
	err := decoder.Decode(&faq)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	errM := UpdateFaq(db, faq)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "FAQ updated."}, http.StatusOK)
}

// DeleteFaq /admin deletes an existing frequently asked question
func DeleteFaq(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "global_admin") {
		return
	}

	faqID := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	errM := RemoveFaq(db, faqID)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "FAQ deleted."}, http.StatusOK)
}

// FindAllFaqs finds and returns all faqs to GetFaqs()
func FindAllFaqs(db *mgo.Database) (faqs []FAQ, errM *Error) {
	return FindFaqsByQuery(db, nil)
}

// FindFaqsByQuery queries the DB and returns faqs collection to FindAllFaqs()
func FindFaqsByQuery(db *mgo.Database, query bson.M) (faqs []FAQ, errM *Error) {
	c := db.C("faqs")
	err := c.Find(query).All(&faqs)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error retrieving faqs: %s\n", err)), Internal: true}
		return
	}

	return
}

func (f *FAQ) Save(db *mgo.Database) *Error {
	c := db.C("faqs")
	_, err := c.UpsertId(f.ID, bson.M{"$set": f})
	if err != nil {
		return &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error saving faq: %s\n",
			err))}
	}

	return nil
}

func UpdateFaq(db *mgo.Database, faq FAQ) *Error {
	c := db.C("faqs")

	// Get old faq
	var oldFaq FAQ
	err := c.FindId(faq.ID).One(&oldFaq)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("The faq you are trying to update does not exist: %s\n", err)), Internal: true}
	}

	// Update faq
	_, err = c.UpsertId(faq.ID, faq)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error updating faq: %s\n", err)), Internal: true}
	}

	return nil
}

func RemoveFaq(db *mgo.Database, id bson.ObjectId) (errM *Error) {
	c := db.C("faqs")
	err := c.RemoveId(id)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error removing FAQ: %s\n", err)), Internal: true}
		return
	}

	return
}
