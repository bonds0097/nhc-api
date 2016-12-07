package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func DBConnect(address string) *mgo.Session {
	ctx := logger.WithField("method", "DBConnect")
	ctx.WithField("address", address).Info("Attempting to connect to mongodb server.")
	session, err := mgo.Dial(address)
	if err != nil {
		panic(err)
	}
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			ctx.WithField("signal", sig).Info("Signal captured - Closing database connection.")
			session.Close()
			os.Exit(1)
		}
	}()

	return session
}

func DBEnsureIndices(s *mgo.Session) (err error) {
	i := mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
		Name:       "email",
	}
	err = s.DB(DBNAME).C("users").EnsureIndex(i)
	if err != nil {
		return
	}

	i = mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		Background: true,
		Name:       "name",
	}

	err = s.DB(DBNAME).C("organizations").EnsureIndex(i)
	if err != nil {
		return
	}

	i = mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		Background: true,
		Name:       "name",
	}

	err = s.DB(DBNAME).C("commitments").EnsureIndex(i)
	if err != nil {
		return
	}

	i = mgo.Index{
		Key:        []string{"code"},
		Unique:     true,
		Background: true,
		Name:       "code",
	}

	err = s.DB(DBNAME).C("families").EnsureIndex(i)
	if err != nil {
		return
	}

	return
}

func DBInit(s *mgo.Session) error {
	ctx := logger.WithField("method", "DBInit")
	ctx.Println("*** Performing Database initialization. ***")
	db := s.DB(DBNAME)

	// Import Organizations
	organizations, err := ioutil.ReadFile(path.Join(APP_DIR, "organizations.json"))
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to read organizations file: %s\n", err))
	}

	var orgs []string
	err = json.Unmarshal(organizations, &orgs)
	if err != nil {
		return errors.New(fmt.Sprintf("Error unmarshalling orgs to JSON: %s\n", err))
	}

	uC := db.C("organizations")
	uC.DropCollection()
	for _, org := range orgs {
		err = uC.Insert(bson.M{"name": org, "needsApproval": false})
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to write organizations to DB: %s\n", err))
		}
	}

	// Import Commitments
	commitments, err := ioutil.ReadFile(path.Join(APP_DIR, "commitments.json"))
	if err != nil {
		ctx.WithError(err).Fatal("Failed to read commitments file.")
	}

	var commits []Commitment
	err = json.Unmarshal(commitments, &commits)
	if err != nil {
		return errors.New(fmt.Sprintf("Error unmarshalling commitments to JSON: %s\n", err))
	}

	uC = db.C("commitments")
	uC.DropCollection()
	for _, commit := range commits {
		commit.ID = bson.NewObjectId()
		err = uC.Insert(commit)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to write commitments to DB: %s\n", err))
		}
	}

	// Initialize Globals
	globals := &Globals{}
	globals.ChallengeStart = time.Date(2016, time.February, 01, 0, 0, 0, 0, time.Local)
	globals.ChallengeEnd = time.Date(2016, time.February, 29, 0, 0, 0, 0, time.Local)
	globals.ChallengeLength = globals.ChallengeEnd.YearDay() - globals.ChallengeStart.YearDay() + 1
	globals.RegistrationOpen = true
	globals.ScorecardEnabled = false

	c := db.C("globals")
	c.DropCollection()
	err = c.Insert(globals)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to write globals to DB: %s\n", err))
	}

	ctx.Println("*** Database initialization complete. ***")

	return nil
	// TODO: Import Resources
}

// Basic data integrity checks and clean-up.
func DBEnsureIntegrity(s *mgo.Session) error {
	ctx := logger.WithField("method", "DBEnsureIntegrity")
	ctx.Println("*** Performing Database integrity checks. ***")
	db := s.DB(DBNAME)

	c := db.C("users")
	// Set all pending users to registered.
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"status": "registered"}},
		ReturnNew: true,
	}
	changeInfo, err := c.Find(bson.M{"status": "pending"}).Apply(change, nil)
	if err != nil && err != mgo.ErrNotFound {
		return errors.New(fmt.Sprintf("Error setting pending users to registered: %s\n", err))
	}

	if changeInfo != nil {
		ctx.WithField("updated", changeInfo.Updated).Info("Updated users from pending to registered.")
	} else {
		ctx.Println("No users updated from pending to registered.")
	}
	// Ensure every participant has a scorecard.
	var registeredUsers []User
	err = c.Find(bson.M{"status": "registered"}).All(&registeredUsers)
	if err != nil {
		return errors.New(fmt.Sprintf("Error retrieving registered users: %s\n", err))
	}
	for _, user := range registeredUsers {
		for index, _ := range user.Participants {
			if user.Participants[index].Scorecard == nil {
				user.Participants[index].Scorecard = GenerateScorecard()
			}
		}
		errM := user.Save(db)
		if errM != nil {
			return errM.Reason
		}
	}

	ctx.Println("*** Database integrity checks complete. ***")
	return nil
}
