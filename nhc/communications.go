package nhc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Message struct {
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Status  []string `json:"status"`
	Roles   []string `json:"roles"`
}

func SendMessage(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, "admin") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var message Message
	err := decoder.Decode(&message)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
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

	// If status selector, body or subject are empty, return an error.
	if len(message.Status) == 0 || message.Body == "" || message.Subject == "" {
		BR(w, r, errors.New(BAD_MESSAGE_ERROR), http.StatusBadRequest)
	}

	// Create main query.
	query := bson.M{"status": bson.M{"$in": message.Status}}

	// If user is a global admin, include the role selector, error out if it is empty.
	if user.Role == GLOBAL_ADMIN.String() || user.Role == GLOBAL_SUPER_ADMIN.String() {
		if len(message.Roles) == 0 {
			BR(w, r, errors.New(BAD_MESSAGE_ERROR), http.StatusBadRequest)
		}
		query["role"] = bson.M{"$in": message.Roles}
	}

	// If user is an org admin, limit to members in same org and only send to org admins and below.
	if user.Role == ORG_ADMIN.String() || user.Role == ORG_SUPER_ADMIN.String() {
		query["organization"] = user.Organization
		query["role"] = bson.M{"$in": []string{USER.String(), ORG_SUPER_ADMIN.String(), ORG_ADMIN.String()}}
	}

	recipients, errM := GetRecipients(db, query)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	go SendBulkMail(recipients, message.Subject, message.Body)

	ServeJSON(w, r, &Response{"status": "Messages sent."}, http.StatusOK)
}

func GetRecipients(db *mgo.Database, query bson.M) (recipients []string, errM *Error) {
	c := db.C("users")
	var users []User
	err := c.Find(query).All(&users)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error finding recipient list: %s\n", err)),
			Internal: true}
	}

	for _, u := range users {
		recipients = append(recipients, u.Email)
	}

	return
}
