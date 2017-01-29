package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

// http handlers

func main() {

	defer msess.Close()

	router := mux.NewRouter()
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", index)
	sub.HandleFunc("/devices", devHandler)

	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler)
	sub.HandleFunc("/custom", customHandler)
	sub.HandleFunc("/login", loginHandler)
	sub.HandleFunc("/logout", logoutHandler)

	// IdleTimeout requires go1.8
	server := http.Server{
		Addr:         ":8082",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		//		IdleTimeout:  120 * time.Second,
		Handler: router,
	}
	fmt.Println("Server started at localhost:8082")
	log.Fatal(server.ListenAndServe())

}
