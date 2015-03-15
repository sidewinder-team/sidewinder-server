package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/anachronistic/apns"
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
	} else {
		return writeJson(insertCode(recordWasCreated), sentJSON, writer)
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

func (self *SidewinderDirector) DeviceMux() *RestEndpoint {
	return (&RestEndpoint{
		Delete: DeviceHandler(self.deleteDevice),
	}).Route("/repositories", RestEndpoint{
		Get:  DeviceHandler(self.GetRepositories),
		Post: DeviceHandler(self.AddRepository),
	}).Route("/notifications", RestEndpoint{
		Post: DeviceHandler(self.PostNotification),
	})
}

func (self *SidewinderDirector) GetRepositories(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	repositories, err := self.Store().RepositoriesForDevice(deviceId)

	if err == nil {
		writeJson(200, repositories, writer)
	}
	return err
}

func (self *SidewinderDirector) AddRepository(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	var repositoryMessage struct{ Name string }
	if decodeErr := json.NewDecoder(request.Body).Decode(&repositoryMessage); decodeErr != nil {
		return decodeErr
	}
	wasInserted, err := self.Store().AddDeviceToRepository(deviceId, repositoryMessage.Name)
	if err != nil {
		return err
	}

	return writeJson(insertCode(wasInserted), repositoryMessage, writer)
}

func insertCode(wasInserted bool) int {
	if wasInserted {
		return 201
	} else {
		return 200
	}
}

func (self *SidewinderDirector) PostNotification(deviceId string, writer http.ResponseWriter, request *http.Request) error {
	var notification map[string]string
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}
	payload := apns.NewPayload()
	payload.Alert = notification["Alert"]

	if err := self.ApnsCommunicator.sendPushNotification(deviceId, payload); err != nil {
		return err
	}
	return writeJson(201, notification, writer)
}

type GithubMessage struct {
	Name    string
	Context string
	State   string
}

func (self *SidewinderDirector) GithubNotify(context web.C, writer http.ResponseWriter, request *http.Request) error {
	var notification GithubMessage
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}

	repository, err := self.Store().FindRepository(notification.Name)
	if err != nil {
		return err
	}

	for _, deviceId := range repository.DeviceList {
		payload := apns.NewPayload()
		payload.Alert = notification.State
		self.ApnsCommunicator.sendPushNotification(deviceId, payload)
	}

	fmt.Fprintf(writer, "Accepted.")
	return nil
}

func (self *SidewinderDirector) CircleNotify(context web.C, writer http.ResponseWriter, request *http.Request) error {
	fmt.Fprintln(os.Stdout, "About to write the recieved header:")
	request.Header.Write(os.Stdout)
	fmt.Fprintln(os.Stdout, "Just wrote the recieved header:")

	var notification map[string]interface{}
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}

	payload, ok := notification["payload"].(map[string]interface{})
	if !ok {
		return errors.New("Sent JSON did not have a 'payload' object.")
	}

	vcsUrl, ok := payload["vcs_url"].(string)
	if !ok {
		return errors.New("Sent JSON did not have a 'vcs_url' string.")
	}

	fmt.Printf("Time to notify everyone registered for project %v\n", vcsUrl)
	return nil
}

func (self *SidewinderDirector) TravisNotify(context web.C, writer http.ResponseWriter, request *http.Request) error {
	fmt.Fprintln(os.Stdout, "About to write the recieved header:")
	request.Header.Write(os.Stdout)
	fmt.Fprintln(os.Stdout, "Just wrote the recieved header:")

	result, err := ioutil.ReadAll(request.Body)
	if err == nil {
		fmt.Printf("Recieved: \n%s\n", result)
	} else {
		fmt.Printf("Err: ", err.Error())
	}

	return nil
}
