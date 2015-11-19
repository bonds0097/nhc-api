package main

import (
	"fmt"
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
)

func init() {
	if p := os.Getenv("PORT"); p != "" {
		PORT = ":" + p
	}
	if m := os.Getenv("MONGODB_URL"); m != "" {
		MONGODB_URL = m
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	dbSession := DBConnect(MONGODB_URL)
	err := DBEnsureIndices(dbSession)
	if err != nil {
		log.Fatalf("Error ensuring DB indices: %s\n", err)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "https://www.nutritionhabitchallenge.com"},
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

	log.Printf("Launching server at http://localhost%s\n", PORT)
	err = http.ListenAndServe(PORT, n)
	if err != nil {
		fmt.Println(err)
	}
}
