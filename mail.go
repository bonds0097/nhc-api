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
	Donation  string
}

const registrationEmail = `
<p>Hi {{.FirstName}},</p>
<p>Congratulations! You are now registered for the Nutrition Habit Challenge 2016. Your participation benefits both you and our community.</p>
{{with .Family}}<p>Here is your family code to share with members of your family, they'll need it when they register: <strong>{{.}}</strong></p>{{end}}
<p>We’ll be sending you an email as we get closer to the event. In the meantime, check out the <a href="https://www.nutritionhabitchallenge.com/resources">Resource Page</a> for great information and insights to help you be successful with the Challenge.</p>
{{if eq .Donation "ysb"}}<p>To donate to the Youth Service Bureau, follow <strong><a href="http://ccysb.com/?page_id=1197" target="_blank">this link</a></strong>.</p>
{{else if eq .Donation "cvim"}}<p>To donate to the Centre Volunteers in Medicine, follow <a href="https://cvim.ejoinme.org/MyPages/CVIMNHC/tabid/524126/Default.aspx" target="_blank">this link</a>.</p>{{end}}
<p><small>If you would like a physical scorecard to track your challenge progress with, download and print the <a href="https://www.nutritionhabitchallenge.com/downloads/scorecard.pdf">PDF scorecard.</a></small></p>
<p>Sincerely,<br />The NHC Team</p>
`

type ResetPasswordTemplate struct {
	FirstName string
	Code      string
}

const resetPasswordEmail = `
<p>Hi {{.FirstName}},<p>
<p>We received a request to reset the password on this account at <a href="https://www.nutritionhabitchallenge.com">https://www.nutritionhabitchallenge.com</a><p>
<p>To reset your password, use the following link: <a href="https://www.nutritionhabitchallenge.com/reset-password/{{.Code}}">https://www.nutritionhabitchallenge.com/reset-password/{{.Code}}</a></p>
<p>If you did not make this request, please ignore this e-mail.</p>
<p>Sincerely,<br />
The NHC Team</p>
`

func SendBulkMail(recipients []string, subject string, body string) (errM *Error) {
	ctx := logger.WithField("method", "SendBulkMail")
	for i := 0; i < len(recipients); i += 500 {
		j := i + 500
		if j > len(recipients) {
			j = len(recipients)
		}
		m := gomail.NewMessage()
		m.SetHeader("From", "do-not-reply@nutritionhabitchallenge.com")
		m.SetHeader("Bcc", recipients[i:j]...)
		m.SetHeader("Subject", subject)
		m.SetBody("text/html", body)

		d := gomail.Dialer{Host: "localhost", Port: MAIL_PORT}
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		if err := d.DialAndSend(m); err != nil {
			ctx.WithError(err).WithField("recipients", j-i).Error("Error sending bulk mail.")
			errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error sending mail: %s\n", err))}
			return
		}
	}

	ctx.WithField("recipients", len(recipients)).WithField("subject", subject).
		Info("Sent bulk mail successfully.")

	return nil
}

func SendMail(recipient string, subject string, body string) (errM *Error) {
	ctx := logger.WithField("method", "SendMail")
	m := gomail.NewMessage()
	m.SetHeader("From", "do-not-reply@nutritionhabitchallenge.com")
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.Dialer{Host: "localhost", Port: MAIL_PORT}
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		ctx.WithError(err).WithField("recipient", recipient).Error("Error sending mail.")
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error sending mail: %s\n", err))}
		return
	}

	ctx.WithField("recipient", recipient).WithField("subject", subject).Info("Successfully sent mail.")

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
	var body bytes.Buffer

	confirmation := RegistrationConfirmationTemplate{FirstName: user.FirstName, Family: user.Family, Donation: user.Donation}
	template := template.Must(template.New("e-mail").Parse(registrationEmail))
	err := template.Execute(&body, &confirmation)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error executing template: %s\n", err)), Internal: true}
		return
	}

	return SendMail(user.Email, "Nutrition Habit Challenge: Registration Confirmation",
		string(body.Bytes()))
}

func SendResetPasswordMail(user *User) (errM *Error) {
	var body bytes.Buffer

	resetPassword := ResetPasswordTemplate{FirstName: user.FirstName, Code: user.ResetCode}
	template := template.Must(template.New("e-mail").Parse(resetPasswordEmail))
	err := template.Execute(&body, &resetPassword)
	if err != nil {
		errM = &Error{Reason: errors.New(fmt.Sprintf("Error executing template: %s\n", err)), Internal: true}
		return
	}

	return SendMail(user.Email, "Nutrition Habit Challenge: Reset Password Request",
		string(body.Bytes()))
}
