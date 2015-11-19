package main

import ()

// Generate a confirmation code for a user.
func GenerateConfirmationCode() (code string) {
	return
}

func ConfirmUser(user *User, code string) (result bool, err Error) {
	// Verify that user's confirmation code and provided string are identical.

	// If strings are identical, change user's status to UNREGISTERED, delete confirmation code from
	// user and return true. Otherwise, return false. Return an error if user does not have a
	// confirmation field.
	return
}
