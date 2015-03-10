package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

type ErrorJson struct {
	Error string
}

func hello(context web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Ahoy, %s!", context.URLParams["name"])
}

func main() {
	apnsCommunicator := NewAPNSCommunicator()
	if err := SetupRoutes("SidewinderMain", apnsCommunicator); err != nil {
		fmt.Errorf("Error on launch:\n%v", err.Error())
		os.Exit(1)
		return
	}

	goji.Serve()
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
		Delete: NewDeviceHandler(sidewinderDirector.deleteDevice),
	}).Handle("/notifications", &RestHandler{
		Post: NewDeviceHandler(sidewinderDirector.PostNotification(apnsComs)),
	})

	return nil
}
