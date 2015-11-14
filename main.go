package main

import (
	"fmt"
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
	DBEnsureIndices(dbSession)

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081", "https://www.nutritionhabitchallenge.com"},
		AllowCredentials: true,
	})

	router := mux.NewRouter().StrictSlash(true)

	// api := router.PathPrefix("/api").Subrouter()

	authApi := router.PathPrefix("/auth").Subrouter()
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

	fmt.Println("Launching server at http://localhost" + PORT)
	err := http.ListenAndServe(PORT, n)
	if err != nil {
		fmt.Println(err)
	}
}
