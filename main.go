package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// http handlers

func main() {

	router := mux.NewRouter()
	sub := router.Host("128.114.97.100").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", devHandler)
	sub.HandleFunc("/{sort}", devHandler)

	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler)

	// IdleTimeout requires go1.8
	server := http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		//		IdleTimeout:  120 * time.Second,
		Handler: router,
	}
	fmt.Println("Server started at localhost:8082")
	log.Fatal(server.ListenAndServe())

}
