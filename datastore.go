package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

type SidewinderDirector struct {
	MongoDB string
	session *mgo.Session
}

func NewSidewinderDirector(mongoDB string) (*SidewinderDirector, error) {
	session, err := mgo.Dial("mongo,localhost")
	if err != nil {
		return nil, err
	}
	return &SidewinderDirector{mongoDB, session}, nil
}

func (self *SidewinderDirector) Store() *SidewinderStore {
	return &SidewinderStore{self.MongoDB, self.session.Copy()}
}

type SidewinderStore struct {
	mongoDB string
	session *mgo.Session
}

func (self *SidewinderStore) DB() *mgo.Database {
	return self.session.DB(self.mongoDB)
}

func (self *SidewinderStore) Close() {
	self.session.Close()
}

type DeviceDocument struct {
	DeviceId string `_id`
}

func (self *SidewinderStore) AddDevice(deviceId string) (bool, error) {
	document := DeviceDocument{deviceId}
	info, err := self.DB().C("devices").UpsertId(deviceId, document)
	switch {
	case err != nil:
		return false, err
	case info.Updated > 0:
		return false, nil
	default:
		return true, nil
	}
}

type DatastoreInfo struct {
	BuildInfo     mgo.BuildInfo
	LiveServers   []string
	DatabaseNames []string
}

func (self *SidewinderDirector) DatastoreInfo(context web.C, writer http.ResponseWriter, request *http.Request) {
	session := self.Store().session

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

	dataStoreInfo := DatastoreInfo{buildInfo, session.LiveServers(), databases}

	err = json.NewEncoder(writer).Encode(&dataStoreInfo)
	if err != nil {
		fmt.Fprintf(writer, "Could not return info from MongoDB.\n%v", err.Error())
		return
	}
}
