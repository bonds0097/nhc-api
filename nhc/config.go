package nhc

import (
	"os"

	mgo "gopkg.in/mgo.v2"

	"github.com/Sirupsen/logrus"
	"github.com/bshuster-repo/logrus-logstash-hook"
)

type Configuration struct {
	FACEBOOK_SECRET string
	GOOGLE_SECRET   string
}

const DBNAME = "nhc"

var config *Configuration

var (
	Logger       *logrus.Logger
	APP_DIR      string
	SignKey      []byte
	VerifyKey    []byte
	GLOBALS      *Globals
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	ENV          string
	DBSession    *mgo.Session
)

func LoadConfig() {
	Logger = logrus.New()
	hook, err := logrus_logstash.NewHook("udp", os.Getenv("LOGSTASH_ADDR"), "nhc-api")
	if err != nil {
		Logger.WithError(err).Warn("Failed to set up logstash hook.")
	} else {
		Logger.Hooks.Add(hook)
	}

	config = &Configuration{
		FACEBOOK_SECRET: os.Getenv("FACEBOOK_SECRET"),
		GOOGLE_SECRET:   os.Getenv("GOOGLE_SECRET"),
	}
}
