package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	// Init Database
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Panicf("Error connecting to database: %s\n", err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	db := session.DB("nhc")

	// Import Organizations
	organizations, err := ioutil.ReadFile("./organizations.json")
	if err != nil {
		log.Fatalf("Failed to read organizations file: %s\n", err)
	}

	var orgs []string
	err = json.Unmarshal(organizations, &orgs)
	if err != nil {
		log.Fatalf("Error unmarshalling to JSON: %s\n", err)
	}

	uC := db.C("organizations")
	uC.DropCollection()
	for _, org := range orgs {
		err = uC.Insert(bson.M{"name": org, "needsApproval": false})
		if err != nil {
			log.Fatalf("Failed to write organizations to DB: %s\n", err)
		}
	}

	// Import Commitments
	type Commitment struct {
		ID    bson.ObjectId `bson:"_id,omitempty" json:"-"`
		Name  string        `bson:"name" json:"name"`
		Links []struct {
			Url   string `bson:"url,omitempty" json:"url,omitempty"`
			Title string `bson:"title,omitempty" json:"title,omitempty"`
		} `bson:"links,omitempty" json:"links,omitempty"`
		Commitments []string `bson:"commitments,omitempty" json:"commitments,omitempty"`
	}

	commitments, err := ioutil.ReadFile("./commitments.json")
	if err != nil {
		log.Fatalf("Failed to read organizations file: %s\n", err)
	}

	var commits []Commitment
	err = json.Unmarshal(commitments, &commits)
	if err != nil {
		log.Fatalf("Error unmarshalling to JSON: %s\n", err)
	}

	uC = db.C("commitments")
	uC.DropCollection()
	for _, commit := range commits {
		err = uC.Insert(commit)
		if err != nil {
			log.Fatalf("Failed to write commitments to DB: %s\n", err)
		}
	}

	// TODO: Import Resources
}
