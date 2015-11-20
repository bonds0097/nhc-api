package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"

	"gopkg.in/gomail.v2"
)

type VerificationTemplate struct {
	FirstName string
	Code      string
}

const verificationEmail = `
<p>Hi {{.FirstName}},<p>
<p>Thank you for creating an account at <a href="https://www.nutritionhabitchallenge.com">https://www.nutritionhabitchallenge.com</a>!<p>
<p>Before you can register, you need to verify your e-mail address.<br />
To do so, just click this link or paste the URL into your browser:<a href="https://www.nutritionhabitchallenge.com/verify/{{.Code}}">https://www.nutritionhabitchallenge.com/verify/{{.Code}}</a></p>
<p>Sincerely,<br />
The NHC Team</p>
`

type RegistrationConfirmationTemplate struct {
	FirstName string
	Family    string
}

func SendMail(recipient string, subject string, body string) (errM *Error) {
	m := gomail.NewMessage()
	m.SetHeader("From", "do-not-reply@nutritionhabitchallenge.com")
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.Dialer{Host: "localhost", Port: MAIL_PORT}
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error sending mail: %s\n", err))}
		return
	}
	return nil
}

func SendVerificationMail(user *User) (errM *Error) {
	var body bytes.Buffer

	confirmation := VerificationTemplate{FirstName: user.FirstName, Code: user.Code}
	template := template.Must(template.New("e-mail").Parse(verificationEmail))
	err := template.Execute(&body, &confirmation)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error executing template: %s\n", err)), Internal: true}
		return
	}

	return SendMail(user.Email, "Nutrition Habit Challenge: E-Mail Verification Required",
		string(body.Bytes()))
}

func SendRegistrationConfirmation(user *User) (errM *Error) {
	return
}
