package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/anachronistic/apns"
	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

var AddDeviceMissingDeviceIdError = ErrorJson{"POST to /devices must be a JSON with a DeviceId property."}

type SidewinderDirector struct {
	MongoDB          string
	session          *mgo.Session
	ApnsCommunicator *APNSCommunicator
	ApiCommunicator  ApiCommunicator
}

func NewSidewinderDirector(mongoDB string, apnsCommunicator *APNSCommunicator, apiCommunicator ApiCommunicator) (*SidewinderDirector, error) {
	session, err := mgo.Dial("mongo,localhost")
	if err != nil {
		return nil, err
	}
	return &SidewinderDirector{mongoDB, session, apnsCommunicator, apiCommunicator}, nil
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

type GithubStatus struct {
	Name        string
	Context     string
	State       string
	Description string
	Branches    []struct {
		Name string
	}
}

func (self *SidewinderDirector) GithubNotify(context web.C, writer http.ResponseWriter, request *http.Request) error {
	var notification GithubStatus
	if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr != nil {
		return decodeErr
	}

	repository, err := self.Store().FindRepository(notification.Name)
	if err != nil {
		return err
	}
	if len(notification.Branches) < 1 {
		return writeJson(400, ErrorJson{"Did not recieve a valid branch in Github status."}, writer)
	}

	branch := notification.Branches[0]
	shouldNotify, err := self.IsFirstSuccessAfterFailure(notification, branch.Name)
	if err != nil {
		return err
	}

	if notification.State == "failure" || notification.State == "error" || shouldNotify {
		payload := apns.NewPayload()
		payload.Alert = notification.Name + ": " + notification.Description
		for _, deviceId := range repository.DeviceList {
			self.ApnsCommunicator.sendPushNotification(deviceId, payload)
		}
	}

	fmt.Fprintf(writer, "Accepted.")
	return nil
}

func (self *SidewinderDirector) getStatusesForCommit(name string, commit string) ([]GithubStatus, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%v/commits/%v/statuses", name, commit)
	response, err := self.ApiCommunicator.Get(url)
	if err != nil {
		return nil, err
	}

	var statuses []GithubStatus
	if decodeErr := json.NewDecoder(response.Body).Decode(&statuses); decodeErr != nil {
		return nil, decodeErr
	}
	return statuses, nil
}

func (self *SidewinderDirector) hasAPreviousFailureInThisCommit(status GithubStatus, branch string) (bool, error) {
	statuses, err := self.getStatusesForCommit(status.Name, branch)
	if err != nil {
		return false, err
	}

	successCount := 0
	for _, status := range statuses {
		switch status.State {
		case "success":
			successCount++
			if successCount == 2 {
				return false, nil
			}
		case "failure":
			if successCount == 1 {
				return true, nil
			}
		case "error":
			if successCount == 1 {
				return true, nil
			}
		}
	}
	return false, nil
}

func (self *SidewinderDirector) hasFailuresInPreviousCommit(status GithubStatus, branch string) (bool, error) {
	previousCommit := branch + "^"
	statuses, err := self.getStatusesForCommit(status.Name, previousCommit)
	if err != nil {
		return false, err
	}

	for _, status := range statuses {
		switch status.State {
		case "success":
			return false, nil
		case "failure":
			return true, nil
		case "error":
			return true, nil
		}
	}
	return false, nil
}

func (self *SidewinderDirector) IsFirstSuccessAfterFailure(status GithubStatus, branch string) (bool, error) {
	if status.State != "success" {
		return false, nil
	}

	previousFailure, err := self.hasAPreviousFailureInThisCommit(status, branch)
	if err != nil {
		return false, err
	} else if previousFailure {
		return true, nil
	} else {
		return self.hasFailuresInPreviousCommit(status, branch)
	}
}
