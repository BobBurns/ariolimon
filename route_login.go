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
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	var b bytes.Buffer
	r.ParseForm()
	currentUser := User{
		Name:     r.FormValue("username"),
		Password: r.FormValue("password"),
	}
	if currentUser.Name == "" && currentUser.Password == "" && r.FormValue("newpass") == "" && r.FormValue("repass") == "" {
		currentUser.IsValid = true
		err := t.ExecuteTemplate(&b, "login.html", currentUser)
		if err != nil {
			fmt.Fprintf(w, "Error with template: %s ", err)
			return
		}
		b.WriteTo(w)
		return
	}
	quser := User{}
	err := dbcoll.Find(bson.M{"name": currentUser.Name}).One(&quser)
	if err != nil {
		errString, _ := err.(*mgo.LastError)
		fmt.Printf("mgo err: %s", errString.Err)
		if errString.Err != "not found" {
			panic(err)
		}
	}

	if quser.Name == "" {
		quser.IsValid = false
	} else {
		passData := []byte(currentUser.Password)
		passString := fmt.Sprintf("%x", sha1.Sum(passData))
		if quser.Name == currentUser.Name && quser.Password == passString {
			if quser.IsFirst == false {
				// TODO handle session cookie
				currentSession = true
				http.Redirect(w, r, "/devices", http.StatusFound)
			}
			newPass := r.FormValue("newpass")
			rePass := r.FormValue("repass")
			if newPass != "" && rePass != "" {
				// TODO store new password in db
				if newPass == rePass {

					updatePassword(newPass)
					currentSession = true
					http.Redirect(w, r, "/devices", http.StatusFound)
				} else {
					quser.Reenter = true
				}
			}
			// first user password is correct --> display new password
			quser.IsValid = true
		} else {
			quser.IsValid = false
		}

	}
	// TODO code to look up user from data base and validate password
	// handle first time sign on
	// handle save password and generate session cookie

	err = t.ExecuteTemplate(&b, "login.html", quser)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}
