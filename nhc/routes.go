package nhc

import (
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func SetupAPI() *negroni.Negroni {
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
	n.Use(DBMiddleware(DBSession))
	n.Use(ParseFormMiddleware())
	n.Use(corsMiddleware)
	n.UseHandler(router)

	return n
}
