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
	apiCommunicator := HttpCommunicator{}
	if err := SetupRoutes("SidewinderMain", apnsCommunicator, apiCommunicator); err != nil {
		fmt.Errorf("Error on launch:\n%v", err.Error())
		os.Exit(1)
		return
	}
	goji.Serve()
}

func SetupRoutes(mongoDB string, apnsComs *APNSCommunicator, apiCommunicator ApiCommunicator) error {
	sidewinderDirector, err := NewSidewinderDirector(mongoDB, apnsComs, apiCommunicator)
	if err != nil {
		return err
	}

	goji.Get("/hello/:name", hello)
	goji.Get("/store/info", RestHandler(sidewinderDirector.DatastoreInfo))

	NewRestMux("/devices", goji.DefaultMux).Use(&RestEndpoint{
		Post: RestHandler(sidewinderDirector.postDevice),
	}).Handle("/:id", sidewinderDirector.DeviceMux())

	hooksMux := NewRestMux("/hooks", goji.DefaultMux)
	hooksMux.Handle("/github", &RestEndpoint{
		Post: RestHandler(sidewinderDirector.GithubNotify)})
	return nil
}
