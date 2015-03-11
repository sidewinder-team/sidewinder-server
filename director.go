package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

var AddDeviceMissingDeviceIdError = ErrorJson{"POST to /devices must be a JSON with a DeviceId property."}

type SidewinderDirector struct {
	MongoDB          string
	session          *mgo.Session
	ApnsCommunicator *APNSCommunicator
}

func NewSidewinderDirector(mongoDB string, communicator *APNSCommunicator) (*SidewinderDirector, error) {
	session, err := mgo.Dial("mongo,localhost")
	if err != nil {
		return nil, err
	}
	return &SidewinderDirector{mongoDB, session, communicator}, nil
}

func (self *SidewinderDirector) Store() *SidewinderStore {
	return &SidewinderStore{self.MongoDB, self.session.Copy()}
}

func (self *SidewinderDirector) DatastoreInfo(context web.C, writer http.ResponseWriter, request *http.Request) error {
	session := self.Store().session

	buildInfo, err := session.BuildInfo()
	if err != nil {
		return fmt.Errorf("Could not connect to MongoDB.\n%v", err.Error())
	}
	writer.Header().Set("Content-Type", "application/json")

	databases, err := session.DatabaseNames()
	if err != nil {
		return fmt.Errorf("Could not retrieve database names.\n%v", err.Error())
	}

	dataStoreInfo := DatastoreInfo{buildInfo, session.LiveServers(), databases}

	return json.NewEncoder(writer).Encode(&dataStoreInfo)
}

func (self *SidewinderDirector) postDevice(context web.C, writer http.ResponseWriter, request *http.Request) error {
	sentJSON := decodeDeviceDocument(request)
	if sentJSON == nil {
		return writeJson(400, AddDeviceMissingDeviceIdError, writer)
	}
	recordWasCreated, err := self.Store().AddDevice(sentJSON.DeviceId)
	if err != nil {
		return err
	} else if recordWasCreated {
		return writeJson(201, sentJSON, writer)
	} else {
		return writeJson(200, sentJSON, writer)
	}
}

func decodeDeviceDocument(request *http.Request) *DeviceDocument {
	var sentJSON DeviceDocument
	if decodeErr := json.NewDecoder(request.Body).Decode(&sentJSON); decodeErr == nil && sentJSON.DeviceId != "" {
		return &sentJSON
	} else {
		return nil
	}
}

type DeviceHandler func(id string, writer http.ResponseWriter, request *http.Request) error

func (self DeviceHandler) ServeHTTPC(context web.C, writer http.ResponseWriter, request *http.Request) {
	deviceId := context.URLParams["id"]
	err := self(deviceId, writer, request)
	if err != nil {
		writeJson(500, ErrorJson{err.Error()}, writer)
	}
}

func (self *SidewinderDirector) deleteDevice(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	result, err := self.Store().FindDevice(deviceId)
	if err != nil {
		return err
	}

	if err := self.Store().DeleteDevice(deviceId); err != nil {
		return err
	}

	return writeJson(200, result, writer)
}

func (self *SidewinderDirector) PostNotification(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	var notification map[string]string
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}

	if err := self.ApnsCommunicator.sendPushNotification(deviceId, notification["Alert"]); err != nil {
		return err
	}
	return writeJson(201, notification, writer)
}

func (self *SidewinderDirector) CircleNotify(context web.C, writer http.ResponseWriter, request *http.Request) error {
	var notification map[string]interface{}
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}

	vcsUrl, ok := notification["vcs_url"].(string)
	if !ok {
		return errors.New("Packet did not have a string property vcs_url")
	}

	fmt.Printf("Time to notify everyone registered for project %v\n", vcsUrl)
	return nil
}
