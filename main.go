package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

type ErrorJson struct {
	Error string
}

var AddDeviceMissingDeviceIdError = ErrorJson{"POST to /devices must be a JSON with a DeviceId property."}

func hello(context web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Herro, %s!", context.URLParams["name"])
}

func main() {
	err := SetupRoutes("SidewinderMain", NewAPNSCommunicator())
	if err == nil {
		goji.Serve()
	} else {
		fmt.Errorf("Error on launch:\n%v", err.Error())
		os.Exit(1)
	}
}

type RestMux struct {
	Mux     *web.Mux
	pattern string
}

func NewRestMux(pattern string, mux *web.Mux) *RestMux {
	return &RestMux{mux, pattern}
}

func (self *RestMux) Use(handler *RestHandler) *RestMux {
	var methods []string
	if handler.Get != nil {
		self.Mux.Get(self.pattern, handler.Get)
		methods = append(methods, "GET")
	}
	if handler.Post != nil {
		self.Mux.Post(self.pattern, handler.Post)
		methods = append(methods, "POST")
	}
	if handler.Put != nil {
		self.Mux.Put(self.pattern, handler.Put)
		methods = append(methods, "PUT")
	}
	if handler.Delete != nil {
		methods = append(methods, "DELETE")
		self.Mux.Delete(self.pattern, handler.Delete)
	}
	if len(methods) > 0 {
		self.Mux.Options(self.pattern, func(context web.C, writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Allow", strings.Join(methods, ","))
		})
	}
	return self
}

func (self *RestMux) Handle(pattern string, handler *RestHandler) *RestMux {
	newRestMux := NewRestMux(self.pattern+pattern, self.Mux)
	newRestMux.Use(handler)
	return newRestMux
}

type RestHandler struct {
	Get     web.HandlerType
	Put     web.HandlerType
	Post    web.HandlerType
	Delete  web.HandlerType
	Options web.HandlerType
}

func SetupRoutes(mongoDB string, apnsComs *APNSCommunicator) error {
	sidewinderDirector, err := NewSidewinderDirector(mongoDB)
	if err != nil {
		return err
	}

	goji.Get("/hello/:name", hello)
	goji.Get("/store/info", sidewinderDirector.DatastoreInfo)

	NewRestMux("/devices", goji.DefaultMux).Use(&RestHandler{
		Post: catchErr(sidewinderDirector.postDevice),
	}).Handle("/:id", &RestHandler{
		Delete: sidewinderDirector.deleteDevice,
	}).Handle("/notifications", NotificationHandler(apnsComs))

	return nil
}

func NotificationHandler(apnsComs *APNSCommunicator) *RestHandler {
	handler := &RestHandler{}
	handler.Post = func(context web.C, writer http.ResponseWriter, request *http.Request) {
		deviceId := context.URLParams["id"]

		var notification map[string]string
		if decodeErr := json.NewDecoder(request.Body).Decode(&notification); decodeErr == nil {
			if err := apnsComs.sendPushNotification(deviceId, notification["Alert"]); err == nil {
				writeJson(201, notification, writer)
			} else {
				writeJson(500, ErrorJson{err.Error()}, writer)
			}
		}
	}
	handler.Options = func(context web.C, writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Allow", "POST")
		writer.WriteHeader(200)
	}

	return handler
}

type Handler func(writer http.ResponseWriter, request *http.Request) error

func catchErr(handler Handler) http.Handler {
	return web.HandlerFunc(func(context web.C, writer http.ResponseWriter, request *http.Request) {
		if err := handler(writer, request); err != nil {
			writeJson(500, ErrorJson{err.Error()}, writer)
		}
	})
}

func (self *SidewinderDirector) postDevice(writer http.ResponseWriter, request *http.Request) error {
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

func (self *SidewinderDirector) deleteDevice(context web.C, writer http.ResponseWriter, request *http.Request) {
	deviceCollection := self.Store().DB().C("devices")
	deviceId := context.URLParams["id"]

	var result DeviceDocument
	if err := deviceCollection.FindId(deviceId).One(&result); err != nil {
		writer.WriteHeader(500)
		fmt.Fprintln(writer, err.Error())
		return
	}

	if err := deviceCollection.RemoveId(deviceId); err != nil {
		writer.WriteHeader(500)
		fmt.Fprintln(writer, err.Error())
		return
	}

	writeJson(200, result, writer)
}

func decodeDeviceDocument(request *http.Request) *DeviceDocument {
	var sentJSON DeviceDocument
	if decodeErr := json.NewDecoder(request.Body).Decode(&sentJSON); decodeErr == nil && sentJSON.DeviceId != "" {
		return &sentJSON
	} else {
		return nil
	}
}

func writeJson(code int, value interface{}, writer http.ResponseWriter) error {
	writer.WriteHeader(code)
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(value)
}
