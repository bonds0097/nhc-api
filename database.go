package main

import (
	"log"
	"os"
	"os/signal"

	"gopkg.in/mgo.v2"
)

func DBConnect(address string) *mgo.Session {
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
			log.Println("%v captured - Closing database connection", sig)
			session.Close()
			os.Exit(1)
		}
	}()

	return session
}

func DBEnsureIndices(s *mgo.Session) error {
	i := mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
		Name:       "email",
	}
	return s.DB(DBNAME).C("users").EnsureIndex(i)
}
