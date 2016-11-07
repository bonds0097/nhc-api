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

type Organization struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	Name          string        `bson:"name" json:"name"`
	NeedsApproval bool          `bson:"needsApproval" json:"needsApproval"`
}

func OrganizationExists(db *mgo.Database, org string) bool {
	c := db.C("organizations")
	count, _ := c.Find(bson.M{"name": org}).Limit(1).Count()
	if count > 0 {
		return true
	} else {
		return false
	}
}

func GetOrganizations(w http.ResponseWriter, r *http.Request) {
	db := GetDB(w, r)
	organizations, errM := FindOrganizations(db)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	b, err := json.Marshal(organizations)
	if err != nil {
		ISR(w, r, errors.New(fmt.Sprintf("Failed to marshal organizations data: %s", err)))
	}
	ServeJSONArray(w, r, string(b), http.StatusOK)
}

func AddOrganization(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, GLOBAL_ADMIN.String()) {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var org Organization
	err := decoder.Decode(&org)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	errM := CreateOrg(db, org.Name, false)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Organization added."}, http.StatusOK)
}

func EditOrganization(w http.ResponseWriter, r *http.Request) {
	// Perform authz check.
	if !IsAuthorized(w, r, GLOBAL_ADMIN.String()) {
		return
	}

	decoder := json.NewDecoder(r.Body)
	var org Organization
	err := decoder.Decode(&org)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	db := GetDB(w, r)
	errM := UpdateOrganization(db, org)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Organization updated."}, http.StatusOK)
}

func DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, GLOBAL_ADMIN.String()) {
		return
	}

	orgID := bson.ObjectIdHex(mux.Vars(r)["id"])

	db := GetDB(w, r)
	errM := RemoveOrganization(db, orgID)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Organization deleted."}, http.StatusOK)
}

func MergeOrganizations(w http.ResponseWriter, r *http.Request) {
	if !IsAuthorized(w, r, GLOBAL_ADMIN.String()) {
		return
	}

	type MergeData struct {
		Organizations []Organization `json:"organizations"`
		NewName       string         `json:"newName"`
	}

	decoder := json.NewDecoder(r.Body)
	var mergeData MergeData
	err := decoder.Decode(&mergeData)
	if err != nil {
		BR(w, r, errors.New(PARSE_ERROR), http.StatusBadRequest)
		return
	}

	if len(mergeData.Organizations) < 1 {
		BR(w, r, errors.New("Merge organizations request missing organizations."), http.StatusBadRequest)
	}

	db := GetDB(w, r)
	errM := MergeOrganizationsInDB(db, mergeData.Organizations, mergeData.NewName)
	if errM != nil {
		HandleModelError(w, r, errM)
		return
	}

	ServeJSON(w, r, &Response{"status": "Organizations successfully merged."}, http.StatusOK)
}

func FindOrganizations(db *mgo.Database) (organizations []Organization, errM *Error) {
	c := db.C("organizations")
	err := c.Find(nil).All(&organizations)
	if err != nil {
		errM = &Error{
			Reason:   errors.New(fmt.Sprintf("Error retrieving organizations from DB: %s", err)),
			Internal: true}
		return
	}

	return
}

func CreateOrg(db *mgo.Database, org string, needsApproval bool) *Error {
	c := db.C("organizations")
	err := c.Insert(bson.M{"_id": bson.NewObjectId(), "name": org, "needsApproval": needsApproval})
	if err != nil && !mgo.IsDup(err) {
		return &Error{Reason: errors.New(fmt.Sprintf("Error creating new org: %s\n", err)), Internal: true}
	}

	return nil
}

func UpdateOrganization(db *mgo.Database, org Organization) *Error {
	c := db.C("organizations")

	// Get old Org so we can propagate change to users that signed up already.
	var oldOrg Organization
	err := c.FindId(org.ID).One(&oldOrg)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("The organization you are trying to update does not exist: %s\n", err)), Internal: true}
	}

	// Update org.
	_, err = c.UpsertId(org.ID, org)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error updating org: %s\n", err)), Internal: true}
	}

	// Propagate change to users.
	uC := db.C("users")
	_, err = uC.UpdateAll(bson.M{"organization": oldOrg.Name}, bson.M{"$set": bson.M{"organization": org.Name}})
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error updating users with new org: %s\n", err)), Internal: true}
	}

	return nil
}

func RemoveOrganization(db *mgo.Database, id bson.ObjectId) *Error {
	c := db.C("organizations")

	// Get old Org so we can propagate change to users that signed up already.
	var oldOrg Organization
	err := c.FindId(id).One(&oldOrg)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("The organization you are trying to delete does not exist: %s\n", err)), Internal: true}
	}

	// Remove organization.
	err = c.RemoveId(id)
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error deleting org: %s\n", err)), Internal: true}
	}

	// Propagate change to users.
	uC := db.C("users")
	_, err = uC.UpdateAll(bson.M{"organization": oldOrg.Name}, bson.M{"$unset": bson.M{"organization": ""}})
	if err != nil {
		return &Error{Reason: errors.New(fmt.Sprintf("Error updating users with new org: %s\n", err)), Internal: true}
	}

	return nil
}

func MergeOrganizationsInDB(db *mgo.Database, orgs []Organization, name string) *Error {
	c := db.C("organizations")

	for index, org := range orgs {
		// Update existing users.
		uC := db.C("users")
		_, err := uC.UpdateAll(bson.M{"organization": org.Name}, bson.M{"$set": bson.M{"organization": name}})
		if err != nil {
			return &Error{Reason: errors.New(fmt.Sprintf("Error updating users with merged org: %s\n", err)), Internal: true}
		}

		if index == 0 {
			// Update first org.
			err = c.UpdateId(org.ID, bson.M{"$set": bson.M{"name": name}})
			if err != nil {
				return &Error{Reason: errors.New(fmt.Sprintf("Error updating merged org name: %s\n", err)), Internal: true}
			}
		} else {
			// Delete remaining orgs.
			err = c.RemoveId(org.ID)
			if err != nil {
				return &Error{Reason: errors.New(fmt.Sprintf("Error deleting org: %s\n", err)), Internal: true}
			}
		}
	}
	return nil
}
