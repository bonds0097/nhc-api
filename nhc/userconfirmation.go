package nhc

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// Generate a confirmation code for a user.
func GenerateConfirmationCode() (code string, errM *Error) {
	c := 32
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error generating user confirmation code: %s\n", err))}
		return
	}

	code = base64.URLEncoding.EncodeToString(b)

	return
}
