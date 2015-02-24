package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

type DatastoreInfo struct {
	BuildInfo     mgo.BuildInfo
	DatabaseNames []string
}

func GetDatastoreInfo(context web.C, writer http.ResponseWriter, request *http.Request) {
	session, err := mgo.Dial("mongo")
	if err != nil {
		fmt.Fprintf(writer, "Could not connect to MongoDB.\n%v", err.Error())
		return
	}
	buildInfo, err := session.BuildInfo()
	if err != nil {
		fmt.Fprintf(writer, "Could not connect to MongoDB.\n%v", err.Error())
		return
	}
	writer.Header().Set("Content-Type", "application/json")

	databases, err := session.DatabaseNames()

	if err != nil {
		fmt.Fprintf(writer, "Could not retrieve database names.\n%v", err.Error())
		return
	}

	dataStoreInfo := DatastoreInfo{buildInfo, databases}

	err = json.NewEncoder(writer).Encode(&dataStoreInfo)
	if err != nil {
		fmt.Fprintf(writer, "Could not return info from MongoDB.\n%v", err.Error())
		return
	}
}
