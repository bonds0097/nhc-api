package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const DBNAME = "nhc"

var (
	PORT        = ":4433"
	MONGODB_URL = "localhost"
	ENV         string
	URL         string
)

func init() {
	if p := os.Getenv("PORT"); p != "" {
		PORT = ":" + p
	}
	if m := os.Getenv("MONGODB_URL"); m != "" {
		MONGODB_URL = m
	}

	flag.StringVar(&ENV, "env", "prod", "Environment to deploy to. Options: prod, test, or dev")
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
	err := DBEnsureIndices(dbSession)
	if err != nil {
		log.Fatalf("Error ensuring DB indices: %s\n", err)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://71.58.99.0:8081", "http://localhost:8081", "https://www.nutritionhabitchallenge.com", "https://test.nutritionhabitchallenge.com"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
	})

	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/commitments", GetCommitments).Methods("GET")
	api.HandleFunc("/organizations", GetOrganizations).Methods("GET")
	api.HandleFunc("/registration", RegisterUser).Methods("POST")

	authApi := router.PathPrefix("/auth").Subrouter()
	authApi.HandleFunc("/", GetAuthStatus).Methods("GET")
	authApi.HandleFunc("/login", Login).Methods("POST")
	authApi.HandleFunc("/signup", SignUp).Methods("POST")
	authApi.HandleFunc("/facebook", LoginWithFacebook).Methods("POST")
	authApi.HandleFunc("/google", LoginWithGoogle).Methods("POST")

	n := negroni.Classic()
	n.Use(JWTMiddleware())
	n.Use(DBMiddleware(dbSession))
	n.Use(ParseFormMiddleware())
	n.Use(corsMiddleware)
	n.UseHandler(router)

	// Start the servers based on whether or not HTTPS is enabled.
	if ENV == "prod" || ENV == "test" {
		log.Println("HTTPS is enabled. Starting server on Port 4433.")

		err := http.ListenAndServeTLS(":4433", "/var/private/nhc_cert.pem", "/var/private/nhc_key.pem", nil)
		if err != nil {
			log.Fatalf("HTTPS Error: %s\n", err)
		}
	} else {
		log.Println("HTTPS is not enabled. Starting server on Port 4433.")

		err := http.ListenAndServe(":4433", nil)
		if err != nil {
			log.Fatalf("HTTPS Error: %s\n", err)
		}
	}
}
