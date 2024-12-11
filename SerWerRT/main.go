package main

import (
	"log"
	"net/http"
)

func main() {
	InitDB()
	defer CloseDB()

	router := setupRouter()

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
