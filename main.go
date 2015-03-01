package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
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
	err := SetupRoutes("SidewinderMain")
	if err == nil {
		goji.Serve()
	} else {
		fmt.Errorf("Error on launch:\n%v", err.Error())
		os.Exit(1)
	}
}

func SetupRoutes(mongoDB string) error {
	sidewinderDirector, err := NewSidewinderDirector(mongoDB)
	if err != nil {
		return err
	}

	goji.Get("/hello/:name", hello)
	goji.Get("/store/info", sidewinderDirector.DatastoreInfo)
	goji.Handle("/devices", handlers.MethodHandler{
		"POST": catchErr(sidewinderDirector.postDevice),
	})
	goji.Options("/devices/:id", func(context web.C, writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Allow", "DELETE")
		writer.WriteHeader(200)
	})
	goji.Delete("/devices/:id", func(context web.C, writer http.ResponseWriter, request *http.Request) {
		deviceCollection := sidewinderDirector.Store().DB().C("devices")
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
	})
	return nil
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
