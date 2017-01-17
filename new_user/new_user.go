package main

import (
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
)

type User struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	IsFirst  bool          `bson:"is_first"`
	Name     string        `bson:"name"`
	Password string        `bson:"password"`
	IsValid  bool
	Error    string
}

func main() {
	var passSum string
	var first bool = true
	// get flags
	user := flag.String("u", "", "user name")
	pass := flag.String("p", "", "password")
	flag.Parse()

	if *user == "" {
		fmt.Println("Username Required!")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *pass == "" { //use default password
		passSum = "4342f8e0e0d10c72434771c211eb2c0478ebe91a"
	} else {
		temp := []byte(*pass)
		passSum = fmt.Sprintf("%x", sha1.Sum(temp))
		first = false
	}

	// init db and add user

	// get db config
	dbData, err := os.Open("configdb.json")
	if err != nil {
		panic(err)
	}
	configdb := struct {
		Host string
		User string
		Pass string
		Db   string
	}{}

	decoder := json.NewDecoder(dbData)
	err = decoder.Decode(&configdb)
	if err != nil {
		panic(err)
	}
	dburl := configdb.User + ":" + configdb.Pass + "@" + configdb.Host + "/" + configdb.Db

	msess, err := mgo.Dial(dburl)
	if err != nil {
		panic(err)
	}
	defer msess.Close()
	msess.SetMode(mgo.Monotonic, true)

	// connection to metric_values
	mcoll := msess.DB("aws_metric_store").C("aws_usr")

	newUser := User{
		Name:     *user,
		Password: passSum,
		IsFirst:  first,
	}
	err = mcoll.Insert(newUser)
	if err != nil {
		panic(err)
	}
	fmt.Printf("User %s successfully added!\n", newUser.Name)

}
