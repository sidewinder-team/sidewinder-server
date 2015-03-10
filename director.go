package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

var AddDeviceMissingDeviceIdError = ErrorJson{"POST to /devices must be a JSON with a DeviceId property."}

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

func (self *SidewinderDirector) postDevice(context web.C, writer http.ResponseWriter, request *http.Request) error {
	sentJSON := decodeDeviceDocument(request)
	if sentJSON == nil {
		return writeJson(400, AddDeviceMissingDeviceIdError, writer)
	} else {
		recordWasCreated, err := self.Store().AddDevice(sentJSON.DeviceId)
		if err != nil {
			return err
		} else if recordWasCreated {
			return writeJson(201, sentJSON, writer)
		} else {
			return writeJson(200, sentJSON, writer)
		}
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

func NewDeviceHandler(deviceHandler DeviceHandler) web.HandlerFunc {
	return func(context web.C, writer http.ResponseWriter, request *http.Request) {
		deviceId := context.URLParams["id"]
		if err := deviceHandler(deviceId, writer, request); err != nil {
			writeJson(500, ErrorJson{err.Error()}, writer)
		}
	}
}

func (self *SidewinderDirector) deleteDevice(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	deviceCollection := self.Store().DB().C("devices")

	var result DeviceDocument
	if err := deviceCollection.FindId(deviceId).One(&result); err != nil {
		return err
	}

	if err := deviceCollection.RemoveId(deviceId); err != nil {
		return err
	}

	return writeJson(200, result, writer)
}

func (self *SidewinderDirector) PostNotification(apnsComs *APNSCommunicator) DeviceHandler {
	return func(deviceId string, writer http.ResponseWriter, request *http.Request) error {
		var notification map[string]string
		if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
			return decodeErr
		}

		if err := apnsComs.sendPushNotification(deviceId, notification["Alert"]); err != nil {
			return err
		}
		return writeJson(201, notification, writer)
	}
}
