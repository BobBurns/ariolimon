package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"time"
)

type User struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	IsFirst  bool          `bson:"is_first"`
	Name     string        `bson:"name"`
	Password string        `bson:"password"`
	IsValid  bool
	Error    string
}

type Session struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Cookie    string        `bson:"cookie"`
	UserName  string        `bson:"username"`
	StartTime int64         `bson:"start_time"`
}

func (sess *Session) DeleteSession() error {
	err := sesscoll.Remove(bson.M{"cookie": sess.Cookie})
	return err
}

func (sess *Session) Check() bool {
	err := sesscoll.Find(bson.M{"cookie": sess.Cookie}).One(sess)
	if err == nil {
		return true
		if debug == 2 {
			fmt.Println("session check!")
		}
	}
	return false
}

var currentSession bool

func (sess *Session) GenerateCookie() {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("Error generating cookie")
	}

	sess.Cookie = base64.URLEncoding.EncodeToString(b)
}
func (sess *Session) Save() error {
	sess.StartTime = time.Now().Unix()
	err := sesscoll.Insert(sess)
	return err

}

func createSession(w http.ResponseWriter, r *http.Request, name string) {

	sess := Session{UserName: name}
	sess.GenerateCookie()
	sess.Save()
	cookie := http.Cookie{
		Name:     "_aricookie",
		Value:    sess.Cookie,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	if debug == 2 {
		fmt.Println("create Session")
	}

}

func webSession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("_aricookie")
	if debug == 2 {
		fmt.Println("cookie ", cookie)
	}
	if err == nil {
		sess := Session{Cookie: cookie.Value}
		if ok := sess.Check(); !ok {
			err = errors.New("Invalid session")
		}
	}
	return err
}

func updatePassword(name, pass string) error {
	if debug == 2 {
		fmt.Println("updatePassword!")
	}
	userq := bson.M{"name": name}
	password := fmt.Sprintf("%x", sha1.Sum([]byte(pass)))
	change := bson.M{"$set": bson.M{"password": password, "is_first": false}}
	err := dbcoll.Update(userq, change)
	if err != nil {
		return errors.New("password Error")
	}
	return nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	redirect := false
	loginUser := User{}
	var b bytes.Buffer
	r.ParseForm()
	userName := r.FormValue("username")
	userPass := r.FormValue("password")
	quser := User{}

	if userName == "" && userPass == "" {
		loginUser.IsValid = true
	} else {

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

				err = updatePassword(userName, newPass)
				if err == nil {
					currentSession = true
					redirect = true
				} else {
					loginUser = User{
						Name:    quser.Name,
						IsValid: false,
						IsFirst: true,
						Error:   "Cannot update Password!",
					}
				}
			} else {
				loginUser = User{
					Name:    quser.Name,
					IsValid: false,
					IsFirst: true,
					Error:   "Passwords must Match!",
				}
			}

		}
	}
	if redirect {
		createSession(w, r, quser.Name)
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
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("_aricookie")
	if err == nil {
		sess := Session{
			Cookie: cookie.Value,
		}
		err = sess.DeleteSession()
		if err == nil || err.Error() == "not found" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	}
	fmt.Println(err)
	http.Redirect(w, r, "/html/error.html", http.StatusInternalServerError)
}

func index(w http.ResponseWriter, r *http.Request) {
	err := webSession(w, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		http.Redirect(w, r, "/devices", http.StatusFound)
	}
}
