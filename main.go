package main

import (
	"flag"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const DBNAME = "nhc"

var (
	logger *logrus.Logger
)

var (
	PORT        string
	MONGODB_URL = "localhost"
	MAIL_PORT   = 25
	ENV         string
	INIT        bool
	APP_DIR     string
	URL         string
	GLOBALS     *Globals
	verifyKey   []byte
	signKey     []byte
	sslCertData []byte
	sslKeyData  []byte
	resetUsers  bool
)

func main() {
	logger = logrus.New()
	hook, err := logrus_logstash.NewHook("udp", os.Getenv("LOGSTASH_ADDR"), "nhc-api")
	if err != nil {
		logger.WithError(err).Warn("Failed to set up logstash hook.")
	} else {
		logger.Hooks.Add(hook)
	}
	ctx := logger.WithFields(logrus.Fields{
		"method": "main",
	})

	ctx.Println("Parsing flags...")
	if m := os.Getenv("MONGODB_URL"); m != "" {
		MONGODB_URL = m
	}

	if s := os.Getenv("MAIL_PORT"); s != "" {
		port, err := strconv.Atoi(s)
		if err != nil {
			ctx.Fatalln("Could not read Mail Port from environment.")
		}
		MAIL_PORT = port
	}

	verifyKey, err = loadPEMBlockFromEnv("JWT_PUB_KEY")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load JWT Verification key.")
	}

	signKey, err = loadPEMBlockFromEnv("JWT_PRIV_KEY")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load JWT Signing key.")
	}

	sslCertData, err = loadPEMBlockFromEnv("SSL_CERT")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load SSL Certificate.")
	}

	sslKeyData, err = loadPEMBlockFromEnv("SSL_KEY")
	if err != nil {
		ctx.WithError(err).Fatal("Failed to load SSL Key.")
	}

	flag.StringVar(&PORT, "port", "8443", "Port to run on.")
	flag.StringVar(&ENV, "env", "prod", "Environment to deploy to. Options: prod, test, or dev")
	flag.BoolVar(&INIT, "init", false, "Initialize the database on startup?")
	flag.StringVar(&APP_DIR, "dir", "/etc/nhc-api/", "Application directory")
	flag.BoolVar(&resetUsers, "reset-users", false, "Reset users to unregistered?")
	flag.Parse()

	if ENV == "prod" {
		URL = "api.nutritionhabitchallenge.com"
	} else if ENV == "test" {
		URL = "test.nutritionhabitchallenge.com"
	} else {
		URL = "localhost"
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	dbSession := DBConnect(MONGODB_URL)

	if INIT {
		err := DBInit(dbSession)
		if err != nil {
			ctx.WithError(err).Fatal("Failed to initialize DB.")
		}
	}

	err = DBEnsureIndices(dbSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB indices.")
	}

	GLOBALS, err = FindGlobals(dbSession.DB(DBNAME))
	if err != nil {
		ctx.Fatalln(err)
	}

	// This has to happen after globals are loaded.
	err = DBEnsureIntegrity(dbSession)
	if err != nil {
		ctx.WithError(err).Fatalf("Error ensuring DB integrity.")
	}

	if resetUsers {
		err = ResetUsers(dbSession)
		if err != nil {
			ctx.WithError(err).Fatal("Error reseting users to unregistered.")
		}
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:900",
			"https://nutritionhabitchallenge.com",
			"https://www.nutritionhabitchallenge.com",
			"https://test.nutritionhabitchallenge.com"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})

	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/globals", GetGlobals).Methods("GET")
	api.HandleFunc("/globals", SaveGlobals).Methods("POST")

	api.HandleFunc("/commitments", GetCommitments).Methods("GET")

	api.HandleFunc("/organizations", GetOrganizations).Methods("GET")
	api.HandleFunc("/admin/organizations", AddOrganization).Methods("POST")
	api.HandleFunc("/admin/organizations", EditOrganization).Methods("PUT")
	api.HandleFunc("/admin/organizations/{id}", DeleteOrganization).Methods("DELETE")
	api.HandleFunc("/admin/organizations/merge", MergeOrganizations).Methods("POST")

	api.HandleFunc("/registration", RegisterUser).Methods("POST")

	api.HandleFunc("/user", UpdateSelf).Methods("PUT")
	api.HandleFunc("/admin/user", GetUsers).Methods("GET")
	api.HandleFunc("/admin/user", EditUser).Methods("PUT")

	api.HandleFunc("/admin/message", SendMessage).Methods("POST")

	api.HandleFunc("/news", FetchNews).Methods("GET")
	api.HandleFunc("/admin/news", ListNews).Methods("GET")
	api.HandleFunc("/admin/news", AddNews).Methods("POST")
	api.HandleFunc("/admin/news/{id}", DeleteNews).Methods("DELETE")
	api.HandleFunc("/admin/news/{id}/publish", PublishNews).Methods("PUT")
	api.HandleFunc("/admin/news/{id}/unpublish", UnpublishNews).Methods("PUT")

	api.HandleFunc("/bonus-question", FetchQuestion).Methods("GET")
	api.HandleFunc("/bonus-question", AnswerQuestion).Methods("POST")
	api.HandleFunc("/admin/bonus-question", GetQuestions).Methods("GET")
	api.HandleFunc("/admin/bonus-question", CreateQuestion).Methods("POST")
	api.HandleFunc("/admin/bonus-question/{id}", DeleteQuestion).Methods("DELETE")
	api.HandleFunc("/admin/bonus-question/{id}/enable", EnableQuestion).Methods("PUT")
	api.HandleFunc("/admin/bonus-question/disable", DisableQuestion).Methods("PUT")

	api.HandleFunc("/participant", GetParticipants).Methods("GET")
	api.HandleFunc("/participant/scorecard", UpdateScorecard).Methods("PUT")
	api.HandleFunc("/admin/participant", GetParticipantsAdmin).Methods("GET")

	api.HandleFunc("/faq", GetFaqs).Methods("GET")
	api.HandleFunc("/admin/faq", AddFaq).Methods("POST")
	api.HandleFunc("/admin/faq", EditFaq).Methods("PUT")
	api.HandleFunc("/admin/faq/{id}", DeleteFaq).Methods("DELETE")

	authAPI := router.PathPrefix("/auth").Subrouter()
	authAPI.HandleFunc("/", GetAuthStatus).Methods("GET")
	authAPI.HandleFunc("/login", Login).Methods("POST")
	authAPI.HandleFunc("/signup", SignUp).Methods("POST")
	authAPI.HandleFunc("/verify", Verify).Methods("POST")
	authAPI.HandleFunc("/verify", ResendVerify).Methods("GET")
	authAPI.HandleFunc("/facebook", LoginWithFacebook).Methods("POST")
	authAPI.HandleFunc("/google", LoginWithGoogle).Methods("POST")
	authAPI.HandleFunc("/password/forgot", ForgotPassword).Methods("POST")
	authAPI.HandleFunc("/password/reset", ResetPassword).Methods("POST")

	n := negroni.Classic()
	n.Use(HeaderMiddleware())
	n.Use(JWTMiddleware())
	n.Use(DBMiddleware(dbSession))
	n.Use(ParseFormMiddleware())
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

	if ENV == "prod" || ENV == "test" {
		// Load SSL Files
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
