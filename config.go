package main

import "os"

type Configuration struct {
	FACEBOOK_SECRET string
	GOOGLE_SECRET   string
}

var config *Configuration

func init() {
	config = &Configuration{
		FACEBOOK_SECRET: os.Getenv("FACEBOOK_SECRET"),
		GOOGLE_SECRET:   os.Getenv("GOOGLE_SECRET"),
	}
}
