package mongo

import (
	//"fmt"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
)

const (
	DB_NAME = "flanders"
)

var connection *mgo.Session

func Connect(connectString string) {
	var err error
	connection, err = mgo.Dial(connectString)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	// Optional. Switch the connection to a monotonic behavior.
	connection.SetMode(mgo.Monotonic, true)

}

func Insert(dbObject interface{}) error {
	collection := connection.DB(DB_NAME).C("message")
	err := collection.Insert(dbObject)
	return err
}
