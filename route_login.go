package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

type User struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	IsFirst  bool          `bson:"is_first"`
	Name     string        `bson:"name"`
	Password string        `bson:"password"`
	IsValid  bool
	Reenter  bool
}

var currentSession bool

func updatePassword(pass string) {
	fmt.Println("updatePassword!")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	redirect := false
	loginUser := User{}
	var b bytes.Buffer
	r.ParseForm()
	userName := r.FormValue("username")
	userPass := r.FormValue("password")

	if userName == "" && userPass == "" {
		loginUser.IsValid = true
	} else {

		quser := User{}
		err := dbcoll.Find(bson.M{"name": userName}).One(&quser)
		if errString, ok := err.(*mgo.LastError); ok {
			//fmt.Printf("mgo err: %s", errString.Err)
			if errString.Err != "not found" {
				panic(err)
			} else {
				loginUser.IsValid = false
			}
		}

		passData := []byte(userPass)
		passString := fmt.Sprintf("%x", sha1.Sum(passData))
		if quser.Name == userName && quser.Password == passString {
			if quser.IsFirst == false {
				// TODO handle session cookie
				currentSession = true
				redirect = true
			} else {
				loginUser.Name = quser.Name
				loginUser.IsValid = true
				loginUser.IsFirst = true
			}
		}
		newPass := r.FormValue("newpass")
		rePass := r.FormValue("repass")

		if newPass != "" && rePass != "" {
			// TODO store new password in db
			if newPass == rePass {

				updatePassword(newPass)
				currentSession = true
				redirect = true
			} else {
				loginUser.Name = quser.Name
				loginUser.Reenter = true
				loginUser.IsValid = false
				loginUser.IsFirst = true
			}

		}
	}
	if redirect {
		http.Redirect(w, r, "/devices", http.StatusFound)
		return
	}
	// first user password is correct --> display new password

	// TODO code to look up user from data base and validate password
	// handle first time sign on
	// handle save password and generate session cookie

	err := t.ExecuteTemplate(&b, "login.html", loginUser)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}
