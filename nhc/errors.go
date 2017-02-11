package nhc

const (
	REQUIRED_ERROR       = "This is a required input."
	PROFANITY_ERROR      = "Please don't use profanity. You're gooder than that."
	BAD_CHOICE_ERROR     = "That is not a valid choice, please select from the available options."
	INTERNAL_ERROR       = "Uh oh, something went wrong on our end. Please try again."
	FAMILY_ERROR         = "The Family Code you entered does not exist. If you did not receive an existing code, leave this field blank."
	ORGANIZATION_ERROR   = "The Organization you entered does not exist, please select from the available options."
	FORBIDDEN_ERROR      = "You are not authorized to access this function."
	MISSING_TOKEN_ERROR  = "Missing Token. Please log in to continue."
	PARSE_ERROR          = "Failed to parse request."
	BAD_MESSAGE_ERROR    = "Message is missing required fields."
	MISSING_FIELDS_ERROR = "Your submissions was missing required fields."
)

var (
	DONATIONS = []string{"ysb", "cvim", "none"}
	SHARING   = []string{"everyone", "none", "organization"}
)
