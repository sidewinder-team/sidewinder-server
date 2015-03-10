package main

import "gopkg.in/mgo.v2"

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
