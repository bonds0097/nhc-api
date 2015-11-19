package main

import (
	"gopkg.in/mgo.v2/bson"
)

type Participant struct {
	ID         bson.ObjectId `bson:"_id,omitempty" json:"-"`
	FirstName  string        `bson:"firstName,omitempty" json:"firstName,omitempty"`
	LastName   string        `bson:"lastName,omitempty" json:"lastName,omitempty"`
	AgeRange   [2]int        `bson:"ageRange,omitempty" json:"ageRange,omitempty"`
	Category   string        `bson:"category,omitempty" json:"category,omitempty"`
	Commitment string        `bson:"commitment,omitempty" json:"commitment,omitempty"`
}
