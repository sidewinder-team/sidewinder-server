package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func hello(context web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Herro, %s!", context.URLParams["name"])
}

func main() {
	SetupRoutes()
	goji.Serve()
}

func SetupRoutes() {
	goji.Get("/hello/:name", hello)
	goji.Get("/store/info", GetDatastoreInfo)

	goji.Handle("/devices", handlers.MethodHandler{
		"POST": http.HandlerFunc(addDevice),
	})
}

func addDevice(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)

	var sentJSON interface{}
	err := decoder.Decode(&sentJSON)
	if err != nil {
		fmt.Fprintln(writer, err.Error())
	}

	encoder := json.NewEncoder(writer)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(201)

	err = encoder.Encode(sentJSON)
	if err != nil {
		fmt.Fprintln(writer, err.Error())
	}
}
