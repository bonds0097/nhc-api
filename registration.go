package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if IsTokenSet(r) {
		tokenData := GetToken(w, r)
		db := GetDB(w, r)

		user, errM := GetUserFromToken(db, tokenData)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		if user.Status == UNCONFIRMED.String() {
			BR(w, r, errors.New("You must confirm your e-mail address before registering."), http.StatusForbidden)
			return
		} else if user.Status == REGISTERED.String() {
			BR(w, r, errors.New("You are already registered."), http.StatusForbidden)
			return
		} else if user.Status != UNREGISTERED.String() {
			BR(w, r, errors.New("You are not allowed to register. Please contact an Administrator."), http.StatusForbidden)
			return
		}

		type RegistrationData struct {
			Organization     string        `json:"organization"`
			Comment          string        `json:"comment"`
			Donation         string        `json:"donation"`
			Sharing          string        `json:"sharing"`
			Participants     []Participant `json:"participants"`
			Family           bool          `json:"family"`
			FamilyCode       string        `json:"familyCode"`
			CustomCommitment bool          `json:"customCommitment,omitempty"`
		}

		decoder := json.NewDecoder(r.Body)
		var registrationData RegistrationData
		err := decoder.Decode(&registrationData)
		if err != nil {
			BR(w, r, errors.New("Could not parse request data."), http.StatusBadRequest)
			log.Printf("Error parsing JSON request: %s\n", err)
			return
		}

		// Validate all data.

		// Generate family code if needed.

		// Change user status appropriately.
		if registrationData.CustomCommitment {
			user.Status = PENDING.String()
		} else {
			user.Status = REGISTERED.String()
		}

		// Save all data.
		user.Organization = registrationData.Organization
		user.Comment = registrationData.Comment
		user.Donation = registrationData.Donation
		user.Sharing = registrationData.Sharing
		user.Participants = registrationData.Participants
		user.Family = registrationData.FamilyCode

		errM = user.Save(db)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		// Send confirmation e-mail.

		ServeJSON(w, r, &Response{"message": "Registration complete."}, http.StatusOK)
		return

	} else {
		BR(w, r, errors.New("Missing Token. Please log in to continue."), http.StatusUnauthorized)
		return
	}
}
