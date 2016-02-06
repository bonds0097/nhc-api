package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const DBNAME = "nhc"

var (
	PORT        string
	MONGODB_URL = "localhost"
	MAIL_PORT   = 25
	ENV         string
	INIT        bool
	APP_DIR     string
	URL         string
	GLOBALS     *Globals
	MAIL_LOG    *log.Logger
)

func init() {
	if m := os.Getenv("MONGODB_URL"); m != "" {
		MONGODB_URL = m
	}

	if s := os.Getenv("MAIL_PORT"); s != "" {
		port, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalln("Could not read Mail Port from environment.")
		}
		MAIL_PORT = port
	}

	flag.StringVar(&PORT, "port", "8443", "Port to run on.")
	flag.StringVar(&ENV, "env", "prod", "Environment to deploy to. Options: prod, test, or dev")
	flag.BoolVar(&INIT, "init", false, "Initialize the database on startup?")
	flag.StringVar(&APP_DIR, "dir", "/etc/nhc-api/", "Application directory")
	flag.Parse()
}

func main() {
	if ENV == "prod" {
		URL = "api.nutritionhabitchallenge.com"
	} else if ENV == "test" {
		URL = "test.nutritionhabitchallenge.com"
	} else {
		URL = "localhost"
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	dbSession := DBConnect(MONGODB_URL)

	// Loggers
	f, err := os.OpenFile("/etc/nhc-api/log/mail.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v\n", err)
	}
	defer f.Close()
	MAIL_LOG = log.New(f, "mail: ", log.LstdFlags)

	if INIT {
		err := DBInit(dbSession)
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = DBEnsureIndices(dbSession)
	if err != nil {
		log.Fatalf("Error ensuring DB indices: %s\n", err)
	}

	GLOBALS, err = FindGlobals(dbSession.DB(DBNAME))
	if err != nil {
		log.Fatalln(err)
	}

	// This has to happen after globals are loaded.
	err = DBEnsureIntegrity(dbSession)
	if err != nil {
		log.Fatalf("Error ensuring DB integrity: %s\n", err)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:8081",
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

	api.HandleFunc("/participant", GetParticipants).Methods("GET")
	api.HandleFunc("/participant/scorecard", UpdateScorecard).Methods("PUT")
	api.HandleFunc("/admin/participant", GetParticipantsAdmin).Methods("GET")

	authApi := router.PathPrefix("/auth").Subrouter()
	authApi.HandleFunc("/", GetAuthStatus).Methods("GET")
	authApi.HandleFunc("/login", Login).Methods("POST")
	authApi.HandleFunc("/signup", SignUp).Methods("POST")
	authApi.HandleFunc("/verify", Verify).Methods("POST")
	authApi.HandleFunc("/verify", ResendVerify).Methods("GET")
	authApi.HandleFunc("/facebook", LoginWithFacebook).Methods("POST")
	authApi.HandleFunc("/google", LoginWithGoogle).Methods("POST")
	authApi.HandleFunc("/password/forgot", ForgotPassword).Methods("POST")
	authApi.HandleFunc("/password/reset", ResetPassword).Methods("POST")

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
		log.Printf("HTTPS is enabled. Starting server on Port %s\n", PORT)
		log.Fatal(s.ListenAndServeTLS("/var/private/nhc_api_cert.pem", "/var/private/nhc_api_key.pem"))
	} else {
		log.Printf("HTTPS is not enabled. Starting server on Port %s\n", PORT)
		log.Fatal(s.ListenAndServe())
	}
}
