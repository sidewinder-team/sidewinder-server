package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"
)

type ErrorJson struct {
	Error string
}

var AddDeviceMissingDeviceIdError = ErrorJson{"POST to /device must be a JSON with a DeviceId property."}

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
		writer.WriteHeader(200)
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

		json.NewEncoder(writer).Encode(result)

	})
	return nil
}

func NewSidewinderDirector(mongoDB string) (*SidewinderDirector, error) {
	session, err := mgo.Dial("mongo,localhost")
	if err != nil {
		return nil, err
	}

	return &SidewinderDirector{mongoDB, session}, nil
}

type Handler func(writer http.ResponseWriter, request *http.Request) error

func catchErr(handler Handler) http.Handler {
	return web.HandlerFunc(func(context web.C, writer http.ResponseWriter, request *http.Request) {
		if err := handler(writer, request); err != nil {
			writer.WriteHeader(500)
			writeJson(struct{ Error string }{err.Error()}, writer)
		}
	})
}

func (self *SidewinderDirector) postDevice(writer http.ResponseWriter, request *http.Request) error {
	var sentJSON DeviceDocument
	if err := json.NewDecoder(request.Body).Decode(&sentJSON); err == nil && sentJSON.DeviceId != "" {
		if recordWasCreated, err := self.Store().AddDevice(sentJSON.DeviceId); err == nil {
			writer.Header().Set("Content-Type", "application/json")

			if recordWasCreated {
				writer.WriteHeader(201)
			} else {
				writer.WriteHeader(200)
			}
			return writeJson(sentJSON, writer)
		} else {
			return err
		}
	}

	writer.WriteHeader(400)
	return writeJson(AddDeviceMissingDeviceIdError, writer)
}

func writeJson(value interface{}, writer http.ResponseWriter) error {
	return json.NewEncoder(writer).Encode(value)
}
