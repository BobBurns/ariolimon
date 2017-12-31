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
	/* change this to IP addr !! */
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", devHandler)
	// gorilla mux var to handle different html outputs
	sub.HandleFunc("/{sort}", devHandler)

	// sd variable for specific service
	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler)

	// IdleTimeout requires go1.8
	server := http.Server{
		Addr:         ":8082",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}
	fmt.Println("Server started at localhost:8082")
	log.Fatal(server.ListenAndServe())

}
