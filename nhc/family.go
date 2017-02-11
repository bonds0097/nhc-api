package nhc

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Family struct {
	ID   bson.ObjectId `bson:"_id"`
	Code string        `bson:"code"`
}

func FamilyExists(db *mgo.Database, code string) bool {
	c := db.C("families")
	count, _ := c.Find(bson.M{"code": code}).Limit(1).Count()
	if count > 0 {
		return true
	} else {
		return false
	}
}

func GenerateFamilyCode(db *mgo.Database, user *User) (code string, errM *Error) {
	// Family code is last name (uppercase) plus 4 digit random number.
	code = CreateCode(user.LastName)
	for FamilyExists(db, code) {
		code = CreateCode(user.Family)
	}

	family := &Family{ID: bson.NewObjectId(), Code: code}

	c := db.C("families")
	err := c.Insert(family)
	if err != nil {
		errM = &Error{Internal: true, Reason: errors.New(fmt.Sprintf("Error creating family code: %s\n", err))}
		return
	}

	return
}

func CreateCode(name string) (code string) {
	rand.Seed(time.Now().Unix())
	return fmt.Sprintf("%s%4d", strings.ToUpper(name), rand.Intn(9999))
}
