package main

const (
	REQUIRED_ERROR   = "This is a required input."
	PROFANITY_ERROR  = "Please don't use profanity. You're gooder than that."
	BAD_CHOICE_ERROR = "That is not a valid choice, please select from the available options."
	INTERNAL_ERROR   = "Uh oh, something went wrong on our end. Please try again."
	FAMILY_ERROR     = "The Family Code you entered does not exist. If you did not receive an existing code, leave this field blank."
)

var (
	DONATIONS = []string{"ysb", "cvim", "none"}
	SHARING   = []string{"everyone", "none", "organization"}
)
