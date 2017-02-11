package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bonds0097/nhc-api/nhc"
)

var (
	PORT        string
	MONGODB_URL = "localhost"
	INIT        bool
	URL         string
	sslCertData []byte
	sslKeyData  []byte
	resetUsers  bool
)

func main() {
	nhc.LoadConfig()

	ctx := nhc.Logger.WithFields(logrus.Fields{
		"method": "main",
	})

	ctx.Println("Parsing flags...")
	if m := os.Getenv("MONGODB_URL"); m != "" {
		MONGODB_URL = m
	}

	nhc.SMTPHost = os.Getenv("SMTP_HOST")

	if s := os.Getenv("SMTP_PORT"); s != "" {
		port, err := strconv.Atoi(s)
		if err != nil {
			ctx.Fatalln("Could not read Mail Port from environment.")
		}
		nhc.SMTPPort = port
	}

	nhc.SMTPUsername = os.Getenv("SMTP_USERNAME")
	nhc.SMTPPassword = os.Getenv("SMTP_PASSWORD")

	var err error
	nhc.VerifyKey, err = loadPEMBlockFromEnv("JWT_PUB_KEY")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load JWT Verification key.")
	}

	nhc.SignKey, err = loadPEMBlockFromEnv("JWT_PRIV_KEY")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load JWT Signing key.")
	}

	flag.StringVar(&PORT, "port", "8443", "Port to run on.")
	flag.StringVar(&nhc.ENV, "env", "prod", "Environment to deploy to. Options: prod, test, or dev")
	flag.BoolVar(&INIT, "init", false, "Initialize the database on startup?")
	flag.StringVar(&nhc.APP_DIR, "dir", "/etc/nhc-api/", "Application directory")
	flag.BoolVar(&resetUsers, "reset-users", false, "Reset users to unregistered?")
	flag.Parse()

	if nhc.ENV == "prod" {
		URL = "api.nutritionhabitchallenge.com"
	} else if nhc.ENV == "test" {
		URL = "test.nutritionhabitchallenge.com"
	} else {
		URL = "localhost"
	}

	nhc.DBSession = nhc.DBConnect(MONGODB_URL)

	if INIT {
		err := nhc.DBInit(nhc.DBSession)
		if err != nil {
			ctx.WithError(err).Fatal("Failed to initialize DB.")
		}
	}

	err = nhc.DBEnsureIndices(nhc.DBSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB indices.")
	}

	nhc.GLOBALS, err = nhc.FindGlobals(nhc.DBSession.DB(nhc.DBNAME))
	if err != nil {
		ctx.Fatalln(err)
	}

	// This has to happen after globals are loaded.
	err = nhc.DBEnsureIntegrity(nhc.DBSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB integrity.")
	}

	if resetUsers {
		err = nhc.ResetUsers(nhc.DBSession)
		if err != nil {
			ctx.WithError(err).Fatal("Error reseting users to unregistered.")
		}
	}

	n := nhc.SetupAPI()

	// Start the servers based on whether or not HTTPS is enabled.
	s := &http.Server{
		Addr:           ":" + PORT,
		Handler:        n,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if nhc.ENV == "prod" || nhc.ENV == "test" {
		// Load SSL Files
		sslCertData, err = loadPEMBlockFromEnv("SSL_CERT")
		if err != nil {
			ctx.WithError(err).Fatal("Failed to load SSL Certificate.")
		}

		sslKeyData, err = loadPEMBlockFromEnv("SSL_KEY")
		if err != nil {
			ctx.WithError(err).Fatal("Failed to load SSL Key.")
		}

		sslCertFile, sslKeyFile, err := loadSSLFiles()
		if err != nil {
			ctx.WithError(err).Fatalf("Failed to load SSL files.")
		}

		ctx.WithField("port", PORT).Info("Starting NHC-API server with HTTPS enabled.")
		ctx.Fatal(s.ListenAndServeTLS(sslCertFile, sslKeyFile))
	} else {
		ctx.WithField("port", PORT).Info("Starting NHC-API server without HTTPS enabled.")
		ctx.Fatal(s.ListenAndServe())
	}
}
