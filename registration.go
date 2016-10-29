package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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
			Organization string        `json:"organization"`
			Comment      string        `json:"comment"`
			Referral     string        `json:"referral"`
			Donation     string        `json:"donation"`
			Sharing      string        `json:"sharing"`
			Participants []Participant `json:"participants"`
			Family       bool          `json:"family"`
			FamilyCode   string        `json:"familyCode"`
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
		type RegistrationValidation struct {
			Organization []string `json:"organization,omitempty"`
			Comment      []string `json:"comment,omitempty"`
			Referral     []string `json:"referral,omitempty"`
			Donation     []string `json:"donation,omitempty"`
			Sharing      []string `json:"sharing,omitempty"`
			Participants []struct {
				FirstName  []string `json:firstName,omitempty"`
				LastName   []string `json:lastName,omitempty"`
				AgeRange   []string `json:ageRange,omitempty"`
				Category   []string `json:category,omitempty"`
				Commitment []string `json:commitment,omitempty"`
			} `json:"participants"`
			FamilyCode []string `json:"familyCode,omitempty"`
		}

		// Assume form is valid and set this to false if needed.
		formIsValid := true
		var registrationValidation RegistrationValidation

		// Check org for profanity.
		if HasProfanity(registrationData.Organization) {
			registrationValidation.Organization = append(registrationValidation.Organization, PROFANITY_ERROR)
			formIsValid = false
		}

		// Check comment for profanity.
		if HasProfanity(registrationData.Comment) {
			registrationValidation.Comment = append(registrationValidation.Comment, PROFANITY_ERROR)
			formIsValid = false
		}
		// Check referral for profanity.
		if HasProfanity(registrationData.Referral) {
			registrationValidation.Referral = append(registrationValidation.Referral, PROFANITY_ERROR)
			formIsValid = false
		}

		// Ensure that donation is a valid selection.
		if registrationData.Donation == "" {
			registrationValidation.Donation = append(registrationValidation.Donation, REQUIRED_ERROR)
			formIsValid = false
		} else if !Contains(DONATIONS, registrationData.Donation) {
			registrationValidation.Donation = append(registrationValidation.Donation, BAD_CHOICE_ERROR)
			formIsValid = false
		}

		// Ensure that sharing is a valid selection.
		if registrationData.Sharing == "" {
			registrationValidation.Sharing = append(registrationValidation.Sharing, REQUIRED_ERROR)
			formIsValid = false
		} else if !Contains(SHARING, registrationData.Sharing) {
			registrationValidation.Sharing = append(registrationValidation.Sharing, BAD_CHOICE_ERROR)
			formIsValid = false
		}

		// Ensure family code exists.
		if registrationData.FamilyCode != "" && !FamilyExists(db, strings.ToUpper(registrationData.FamilyCode)) {
			registrationValidation.FamilyCode = append(registrationValidation.FamilyCode, FAMILY_ERROR)
			formIsValid = false
		}

		// At this point, if we have any validation issues, return the validation struct.
		if !formIsValid {
			b, err := json.Marshal(registrationValidation)
			if err != nil {
				ISR(w, r, errors.New(fmt.Sprintf("Error marshaling registration validation: %s\n", err)))
				return
			}
			parse := &Response{}
			json.Unmarshal(b, parse)
			ServeJSON(w, r, parse, http.StatusBadRequest)
			return
		}

		// Try and create new org. We don't worry about dup errors.
		errM = CreateOrg(db, registrationData.Organization, true)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		// Save all data.
		user.Organization = registrationData.Organization
		user.Comment = registrationData.Comment
		user.Referral = registrationData.Referral
		user.Donation = registrationData.Donation
		user.Sharing = registrationData.Sharing
		user.Participants = registrationData.Participants

		// Generate family code if needed.
		if registrationData.Family && registrationData.FamilyCode == "" {
			familyCode, errM := GenerateFamilyCode(db, user)
			if errM != nil {
				HandleModelError(w, r, errM)
				return
			}
			user.Family = familyCode
		} else {
			user.Family = registrationData.FamilyCode
		}

		// Create ID for each participant. Set points and create empty scorecard.
		for key, _ := range user.Participants {
			user.Participants[key].ID = key
			user.Participants[key].Points = 0
			user.Participants[key].Scorecard = GenerateScorecard()
		}

		// Change user status appropriately.
		user.Status = REGISTERED.String()

		errM = user.Save(db)
		if errM != nil {
			HandleModelError(w, r, errM)
			return
		}

		// Send confirmation e-mail.
		go SendRegistrationConfirmation(user)

		ServeJSON(w, r, &Response{"message": "Registration complete."}, http.StatusOK)
		return

	} else {
		BR(w, r, errors.New("Missing Token. Please log in to continue."), http.StatusUnauthorized)
		return
	}
}
