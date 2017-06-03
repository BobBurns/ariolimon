package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//query amazon and display device statistics
func devHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stemp := vars["sort"]

	// getThresholds function in utils.go
	// loads predefined thresholds from thresh.json
	err, querys := getThresholds()
	if err != nil {
		log.Printf("Error with getStatistics: %s", err)
		http.Redirect(w, r, "/html/error.html", http.StatusFound)
	}

	// get statistics for every metric defined in thresh.json
	for i, _ := range querys {
		err = querys[i].getStatistics("-10m")

		if err != nil {
			log.Printf("Error with getStatistics: %s", err)
			http.Redirect(w, r, "/html/error.html", http.StatusFound)

		}
	}
	var b bytes.Buffer

	// display by stemp var in url
	h := "home2.html"
	switch stemp {
	case "crit":
		h = "crit.html"
	case "warn":
		h = "warn.html"
	case "ok":
		h = "ok.html"
	}
	err = t.ExecuteTemplate(&b, h, querys)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}
