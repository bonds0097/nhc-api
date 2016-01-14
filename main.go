package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const DBNAME = "nhc"

var (
	PORT        = ":4433"
	MONGODB_URL = "localhost"
	MAIL_PORT   = 25
	ENV         string
	INIT        bool
	APP_DIR     string
	URL         string
	GLOBALS     *Globals
)

func init() {
	if p := os.Getenv("PORT"); p != "" {
		PORT = ":" + p
	}
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

	flag.StringVar(&ENV, "env", "prod", "Environment to deploy to. Options: prod, test, or dev")
	flag.BoolVar(&INIT, "init", false, "Initialize the database on startup?")
	flag.StringVar(&APP_DIR, "dir", "/etc/nhc-api/", "Application directory")
	flag.Parse()
}

func main() {
	if ENV == "prod" {
		URL = "www.nutritionhabitchallenge.com"
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
			log.Fatalln(err)
		}
	}

	err := DBEnsureIndices(dbSession)
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
			"https://www.nutritionhabitchallenge.com",
			"https://test.nutritionhabitchallenge.com"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})

	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/globals", GetGlobals).Methods("GET")

	api.HandleFunc("/commitments", GetCommitments).Methods("GET")

	api.HandleFunc("/organizations", GetOrganizations).Methods("GET")
	api.HandleFunc("/admin/organizations", EditOrganization).Methods("PUT")
	api.HandleFunc("/admin/organizations/{id}", DeleteOrganization).Methods("DELETE")
	api.HandleFunc("/admin/organizations/merge", MergeOrganizations).Methods("POST")

	api.HandleFunc("/registration", RegisterUser).Methods("POST")

	api.HandleFunc("/admin/user", GetUsers).Methods("GET")
	api.HandleFunc("/admin/user", EditUser).Methods("PUT")

	api.HandleFunc("/participant", GetParticipants).Methods("GET")
	api.HandleFunc("/participant/{id}/scorecard", UpdateScorecard).Methods("PUT")
	api.HandleFunc("/admin/participant", GetParticipantsAdmin).Methods("GET")

	authApi := router.PathPrefix("/auth").Subrouter()
	authApi.HandleFunc("/", GetAuthStatus).Methods("GET")
	authApi.HandleFunc("/login", Login).Methods("POST")
	authApi.HandleFunc("/signup", SignUp).Methods("POST")
	authApi.HandleFunc("/verify", Verify).Methods("POST")
	authApi.HandleFunc("/verify", ResendVerify).Methods("GET")
	authApi.HandleFunc("/facebook", LoginWithFacebook).Methods("POST")
	authApi.HandleFunc("/google", LoginWithGoogle).Methods("POST")

	n := negroni.Classic()
	n.Use(HeaderMiddleware())
	n.Use(JWTMiddleware())
	n.Use(DBMiddleware(dbSession))
	n.Use(ParseFormMiddleware())
	n.Use(corsMiddleware)
	n.UseHandler(router)

	// Start the servers based on whether or not HTTPS is enabled.
	if ENV == "prod" || ENV == "test" {
		log.Println("HTTPS is enabled. Starting server on Port 4433.")

		err := http.ListenAndServeTLS(":4433", "/var/private/nhc_cert.pem", "/var/private/nhc_key.pem", n)
		if err != nil {
			log.Fatalf("HTTPS Error: %s\n", err)
		}
	} else {
		log.Println("HTTPS is not enabled. Starting server on Port 4433.")

		err := http.ListenAndServe(":4433", n)
		if err != nil {
			log.Fatalf("HTTPS Error: %s\n", err)
		}
	}
}
