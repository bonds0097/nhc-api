package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"time"

	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/bonds0097/nhc-api/nhc"
)

var fname, mongoURL string
var dryRun bool

func main() {
	flag.StringVar(&fname, "file", "", "csv file to load users from")
	flag.StringVar(&mongoURL, "mongo", "mongo", "mongoDB URL")
	flag.BoolVar(&dryRun, "dry-run", true, "if false, changes will be made to db")
	flag.Parse()

	if fname == "" {
		logrus.Fatal("A file to load users from is required.")
	}

	// Get SMTP Vars
	nhc.SMTPHost = os.Getenv("SMTP_HOST")
	nhc.SMTPPort = 587
	nhc.SMTPUsername = os.Getenv("SMTP_USERNAME")
	nhc.SMTPPassword = os.Getenv("SMTP_PASSWORD")

	// Load CSV File
	f, err := os.Open(fname)
	if err != nil {
		logrus.Fatalf("Failed to open file with users: %s", err)
	}

	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Turn records into users
	//
	// Columns of interest:
	// 1: First Name
	// 2: Last Name
	// 4: Family Code
	// 5: E-Mail Address
	// 6: Organization
	// 7-53: Commitments
	var prevEmail string
	var users []nhc.User
	var u nhc.User
	var pCount int
	for i, record := range records[1:] {
		curEmail := strings.ToLower(strings.TrimSpace(record[5]))

		// If the current e-mail differs from the previous e-mail, treat as
		// new user, otherwise participant of current user.
		if curEmail != prevEmail {
			// Append old user to list of users
			users = append(users, u)

			// New Empty User
			u = nhc.User{}

			// Add User Data
			u.Password = nhc.RandToken()
			u.Email = curEmail
			u.FirstName = record[1]
			u.LastName = record[2]
			u.Family = record[4]
			u.Organization = record[6]
			u.Role = nhc.USER.String()
			u.Status = nhc.REGISTERED.String()
			u.CreatedOn = time.Now()
			u.LastLogin = time.Now()

			// Add Participant Data
			p := nhc.Participant{}
			p.ID = 0
			p.FirstName = u.FirstName
			p.LastName = u.LastName
			p.Category = "Other"
			p.Commitment = findCommitment(record[7:54])

			u.Participants = append(u.Participants, p)
			pCount++

			// If this is the last record, flush as well.
			if i == len(records)-1 {
				users = append(users, u)
			}

			prevEmail = curEmail

			continue
		}

		// Add Participant Data to Existing User
		p := nhc.Participant{}
		p.ID = len(u.Participants)
		p.FirstName = record[1]
		p.LastName = record[2]
		p.Category = "Other"
		p.Commitment = findCommitment(record[7:54])

		u.Participants = append(u.Participants, p)
		pCount++

	}

	logrus.Infof("Parsed %d users with a total of %d participants.", len(users), pCount)

	if !dryRun {
		var sucCount, errCount int
		var sUsers []nhc.User
		nhc.LoadConfig()
		// Update DB
		s := nhc.DBConnect(mongoURL)
		if s == nil {
			logrus.Fatalf("Failed to connect to mongoDB.")
		}
		err := nhc.DBEnsureIndices(s)
		if err != nil {
			logrus.Fatalf("Failed to ensure indices on mongo: %s", err)
		}
		db := s.DB(nhc.DBNAME)

		before, err := db.C("users").Count()
		if err != nil {
			logrus.Fatalf("Failed to get count from users collection: %s", err)
		}
		logrus.Infof("Current user count: %d.", before)
		logrus.Info("Adding users to database...")
		for _, u := range users {
			errM := nhc.CreateUser(db, &u)
			if errM != nil {
				logrus.Errorf("Failed to add user %s to database: %s", u.Email, errM.Reason)
				errCount++
				continue
			}
			sucCount++
			sUsers = append(sUsers, u)
		}

		logrus.Infof("Users added: %d. Failed: %d.", sucCount, errCount)
		after, err := db.C("users").Count()
		if err != nil {
			logrus.Fatalf("Failed to get count from users collection: %s", err)
		}
		logrus.Infof("User count after operation: %d.", after)

		// Send reset password e-mail to each user
		if nhc.SMTPHost == "" {
			logrus.Info("Skipping sending password reset e-mails, no SMTP Host set.")
			return
		}

		logrus.Info("Sending password reset e-mails.")
		var rpSucCount, rpErrCount int
		for _, u := range sUsers {
			// Generate code and save in DB. Then send email to user.
			code, _ := nhc.GenerateConfirmationCode()
			u.ResetCode = code
			errM := u.Save(db)
			if errM != nil {
				logrus.Errorf("Failed to set reset code for user %s: %s.", u.Email, errM.Reason)
				rpErrCount++
				continue
			}

			errM = nhc.SendResetPasswordMail(&u)
			if errM != nil {
				logrus.Errorf("Failed to send reset password e-mail for user %s: %s.", u.Email, errM.Reason)
				rpErrCount++
				continue
			}
			rpSucCount++
		}
		logrus.Infof("Successful reset pw e-mails: %d. Failed: %d", rpSucCount, rpErrCount)
	}
}

func findCommitment(fields []string) string {
	for _, field := range fields {
		if field != "" {
			return field
		}
	}

	return "Unknown Commitment"
}
