package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

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

func (self *SidewinderStore) FindDevice(deviceId string) (DeviceDocument, error) {
	deviceCollection := self.DB().C("devices")

	var result DeviceDocument
	err := deviceCollection.FindId(deviceId).One(&result)
	return result, err
}

func (self *SidewinderStore) DeleteDevice(deviceId string) error {
	deviceCollection := self.DB().C("devices")
	return deviceCollection.RemoveId(deviceId)
}

type RepositoryDocument struct {
	Name string `_id`
}

func (self *SidewinderStore) AddDeviceToRepository(devideId, repositoryName string) error {
	repositoryCollection := self.DB().C("repositories")
	update := struct {
		Push interface{} `$push`
	}{struct{ DeviceList string }{devideId}}

	_, err := repositoryCollection.UpsertId(repositoryName, update)

	return err
}

func (self *SidewinderStore) RepositoriesForDevice(deviceId string) ([]RepositoryDocument, error) {
	repositoryCollection := self.DB().C("repositories")

	queryData := struct {
		DeviceList string
	}{deviceId}

	query := repositoryCollection.Find(queryData)
	result := make([]RepositoryDocument, 0)
	err := query.Select(bson.M{"_id": 1}).All(&result)
	return result, err
}

type DatastoreInfo struct {
	BuildInfo     mgo.BuildInfo
	LiveServers   []string
	DatabaseNames []string
}
