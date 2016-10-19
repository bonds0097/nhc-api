package main

import (
	"encoding/json"
	// "log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

const (
	PROFANITY_URL = "http://www.wdyl.com/profanity?q="
)

func HasProfanity(input string) bool {
	return QueryGoogleProfanityFilter(AddSpaces(input))
}

func AddSpaces(input string) (output string) {
	re := regexp.MustCompile(`\W+`)
	return re.ReplaceAllString(input, " ")
}

func QueryGoogleProfanityFilter(query string) (isProfane bool) {
	ctx := logger.WithField("method", "QueryGoogleProfanityFilter")
	type ApiResponse struct {
		Response string `json:"response"`
	}

	resp, err := http.Get(PROFANITY_URL + url.QueryEscape(query))
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error querying Google Profanity Filter.")
		// log.Printf("Error querying Google Profanity Filter, look into it: %s\n", err)
		return false
	}
	decoder := json.NewDecoder(resp.Body)

	var response ApiResponse
	err = decoder.Decode(&response)
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error parsing Google's response.")
		// log.Printf("Error parsing Google's response, look into it: %s\n", err)
		return false
	}

	isProfane, err = strconv.ParseBool(response.Response)
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error parsing Google's response.")
		// log.Printf("Error parsing Google's response, look into it: %s\n", err)
		return false
	}

	return
}
