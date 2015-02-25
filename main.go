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
		"POST": web.HandlerFunc(sidewinderDirector.addDevice),
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

func (self *SidewinderDirector) addDevice(context web.C, writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)

	var sentJSON map[string]interface{}
	err := decoder.Decode(&sentJSON)
	if err != nil {
		writer.WriteHeader(500)
		fmt.Fprintln(writer, err.Error())
	}

	deviceId := sentJSON["DeviceId"]
	err = self.Store().AddDevice(deviceId.(string))

	if err != nil {
		writer.WriteHeader(500)
		fmt.Fprintln(writer, err.Error())
	}

	encoder := json.NewEncoder(writer)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(201)

	err = encoder.Encode(sentJSON)
	if err != nil {
		writer.WriteHeader(500)
		fmt.Fprintln(writer, err.Error())
	}
}
