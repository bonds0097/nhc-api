package nhc

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

const (
	PROFANITY_URL = "http://www.wdyl.com/profanity?q="
)

func HasProfanity(input string) bool {
	return false
}

func AddSpaces(input string) (output string) {
	re := regexp.MustCompile(`\W+`)
	return re.ReplaceAllString(input, " ")
}

func QueryGoogleProfanityFilter(query string) (isProfane bool) {
	ctx := Logger.WithField("method", "QueryGoogleProfanityFilter")
	type ApiResponse struct {
		Response string `json:"response"`
	}

	resp, err := http.Get(PROFANITY_URL + url.QueryEscape(query))
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error querying Google Profanity Filter.")
		return false
	}
	decoder := json.NewDecoder(resp.Body)

	var response ApiResponse
	err = decoder.Decode(&response)
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error parsing Google's response.")
		return false
	}

	isProfane, err = strconv.ParseBool(response.Response)
	if err != nil {
		ctx.WithError(err).WithField("error", err).Error("Error parsing Google's response.")
		return false
	}

	return
}
