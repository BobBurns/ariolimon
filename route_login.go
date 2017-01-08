package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

type User struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	IsFirst  bool          `bson:"is_first"`
	Name     string        `bson:"name"`
	Password string        `bson:"password"`
	IsValid  bool
	Cookie   string
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	currentUser := User{
		Name:     r.FormValue("username"),
		Password: r.FormValue("password"),
	}
	if currentUser.Name == "" {
		currentUser.isValid == true
	} else {
		// TODO code to look up user from data base and validate password
		// handle first time sign on
		// handle save password and generate session cookie
	}

	var b bytes.Buffer
	err := t.ExecuteTemplate(&b, "login.html", currentUser)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}
