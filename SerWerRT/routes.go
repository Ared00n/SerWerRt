package main

import (
	"github.com/gorilla/mux"
)

func setupRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/register", registerHandler)
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc("/personal_cabinet", personalCabinetHandler)
	router.HandleFunc("/uslugi", uslugi)
	router.HandleFunc("/candidates", candidatesHandler)
	router.HandleFunc("/register_candidate", registerCandidateHandler)

	router.HandleFunc("/works", worksHandler).Methods("GET")
	router.HandleFunc("/works/add", addWorkHandler).Methods("POST")
	router.HandleFunc("/works/delete", deleteWorkHandler).Methods("DELETE")
	router.HandleFunc("/works/update", updateWorkHandler).Methods("POST")
	router.HandleFunc("/works/{id}", getWorkHandler).Methods("GET")

	return router
}
