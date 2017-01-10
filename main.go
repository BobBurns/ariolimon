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

	// map servicenames and hostnames for http handlers
	templateService := Services{}
	var namemap = make(map[string]MetricQuery)
	for i, _ := range hosts {
		namemap[hosts[i].Name] = hosts[i]
		templateService.Service = append(templateService.Service, hosts[i].Name)
	}

	router := mux.NewRouter()
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", devHandler(hosts))
	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler(namemap))
	sub.HandleFunc("/custom", customHandler(templateService))
	sub.HandleFunc("/login", loginHandler)

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
