package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bonds0097/nhc-api/nhc"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

	dbSession := nhc.DBConnect(MONGODB_URL)

	if INIT {
		err := nhc.DBInit(dbSession)
		if err != nil {
			ctx.WithError(err).Fatal("Failed to initialize DB.")
		}
	}

	err = nhc.DBEnsureIndices(dbSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB indices.")
	}

	nhc.GLOBALS, err = nhc.FindGlobals(dbSession.DB(nhc.DBNAME))
	if err != nil {
		ctx.Fatalln(err)
	}

	// This has to happen after globals are loaded.
	err = nhc.DBEnsureIntegrity(dbSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB integrity.")
	}

	if resetUsers {
		err = nhc.ResetUsers(dbSession)
		if err != nil {
			ctx.WithError(err).Fatal("Error reseting users to unregistered.")
		}
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:9000",
			"https://nutritionhabitchallenge.com",
			"https://www.nutritionhabitchallenge.com",
			"https://test.nutritionhabitchallenge.com"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})

	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/globals", nhc.GetGlobals).Methods("GET")
	api.HandleFunc("/globals", nhc.SaveGlobals).Methods("POST")

	api.HandleFunc("/commitments", nhc.GetCommitments).Methods("GET")

	api.HandleFunc("/organizations", nhc.GetOrganizations).Methods("GET")
	api.HandleFunc("/admin/organizations", nhc.AddOrganization).Methods("POST")
	api.HandleFunc("/admin/organizations", nhc.EditOrganization).Methods("PUT")
	api.HandleFunc("/admin/organizations/{id}", nhc.DeleteOrganization).Methods("DELETE")
	api.HandleFunc("/admin/organizations/merge", nhc.MergeOrganizations).Methods("POST")

	api.HandleFunc("/registration", nhc.RegisterUser).Methods("POST")

	api.HandleFunc("/user", nhc.UpdateSelf).Methods("PUT")
	api.HandleFunc("/admin/user", nhc.GetUsers).Methods("GET")
	api.HandleFunc("/admin/user", nhc.EditUser).Methods("PUT")

	api.HandleFunc("/admin/message", nhc.SendMessage).Methods("POST")

	api.HandleFunc("/news", nhc.FetchNews).Methods("GET")
	api.HandleFunc("/admin/news", nhc.ListNews).Methods("GET")
	api.HandleFunc("/admin/news", nhc.AddNews).Methods("POST")
	api.HandleFunc("/admin/news/{id}", nhc.DeleteNews).Methods("DELETE")
	api.HandleFunc("/admin/news/{id}/publish", nhc.PublishNews).Methods("PUT")
	api.HandleFunc("/admin/news/{id}/unpublish", nhc.UnpublishNews).Methods("PUT")

	api.HandleFunc("/bonus-question", nhc.FetchQuestion).Methods("GET")
	api.HandleFunc("/bonus-question", nhc.AnswerQuestion).Methods("POST")
	api.HandleFunc("/admin/bonus-question", nhc.GetQuestions).Methods("GET")
	api.HandleFunc("/admin/bonus-question", nhc.CreateQuestion).Methods("POST")
	api.HandleFunc("/admin/bonus-question/{id}", nhc.DeleteQuestion).Methods("DELETE")
	api.HandleFunc("/admin/bonus-question/{id}/enable", nhc.EnableQuestion).Methods("PUT")
	api.HandleFunc("/admin/bonus-question/disable", nhc.DisableQuestion).Methods("PUT")

	api.HandleFunc("/participant", nhc.GetParticipants).Methods("GET")
	api.HandleFunc("/participant/scorecard", nhc.UpdateScorecard).Methods("PUT")
	api.HandleFunc("/admin/participant", nhc.GetParticipantsAdmin).Methods("GET")

	api.HandleFunc("/faq", nhc.GetFaqs).Methods("GET")
	api.HandleFunc("/admin/faq", nhc.AddFaq).Methods("POST")
	api.HandleFunc("/admin/faq", nhc.EditFaq).Methods("PUT")
	api.HandleFunc("/admin/faq/{id}", nhc.DeleteFaq).Methods("DELETE")

	authAPI := router.PathPrefix("/auth").Subrouter()
	authAPI.HandleFunc("/", nhc.GetAuthStatus).Methods("GET")
	authAPI.HandleFunc("/login", nhc.Login).Methods("POST")
	authAPI.HandleFunc("/signup", nhc.SignUp).Methods("POST")
	authAPI.HandleFunc("/verify", nhc.Verify).Methods("POST")
	authAPI.HandleFunc("/verify", nhc.ResendVerify).Methods("GET")
	authAPI.HandleFunc("/facebook", nhc.LoginWithFacebook).Methods("POST")
	authAPI.HandleFunc("/google", nhc.LoginWithGoogle).Methods("POST")
	authAPI.HandleFunc("/password/forgot", nhc.ForgotPassword).Methods("POST")
	authAPI.HandleFunc("/password/reset", nhc.ResetPassword).Methods("POST")

	n := negroni.Classic()
	n.Use(nhc.HeaderMiddleware())
	n.Use(nhc.JWTMiddleware())
	n.Use(nhc.DBMiddleware(dbSession))
	n.Use(nhc.ParseFormMiddleware())
	n.Use(corsMiddleware)
	n.UseHandler(router)

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
