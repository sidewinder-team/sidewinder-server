package main_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	server "github.com/sidewinder-team/sidewinder-server"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"gopkg.in/mgo.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	TestDatabaseName = "SidewinderTest"
)

func NewPOSTRequestWithJSON(path string, body interface{}) (*http.Request, []byte) {
	data, err := json.Marshal(body)
	Expect(err).NotTo(HaveOccurred())

	request, err := http.NewRequest("POST", path, bytes.NewReader(data))
	Expect(err).NotTo(HaveOccurred())
	return request, data
}

func NewRequest(method string, path string) *http.Request {
	request, err := http.NewRequest(method, path, nil)
	Expect(err).NotTo(HaveOccurred())
	return request
}

var _ = Describe("Endpoint", func() {

	Describe("/devices", func() {

		var db *mgo.Database

		BeforeEach(func() {
			server.SetupRoutes(TestDatabaseName)

			session, err := mgo.Dial("mongo,localhost")
			Expect(err).NotTo(HaveOccurred())
			db = session.DB(TestDatabaseName)
			Expect(db).NotTo(BeNil())

			db.C("devices").DropCollection()
		})

		AfterEach(func() {
			goji.DefaultMux = web.New()
		})

		Describe("POST", func() {
			It("is able to successfully add a new device.", func() {
				responseRecorder := httptest.NewRecorder()
				deviceInfo := server.DeviceDocument{"abracadabra"}

				request, data := NewPOSTRequestWithJSON("/devices", deviceInfo)
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(201))
				Expect(responseRecorder.Body.String()).To(MatchJSON(data))

				deviceCollection := db.C("devices")
				var result []server.DeviceDocument
				deviceCollection.FindId(deviceInfo.DeviceId).All(&result)
				Expect(result).To(Equal([]server.DeviceDocument{deviceInfo}))
			})
		})

		Describe("OPTIONS", func() {
			It("Lists all the provided functions.", func() {
				request, err := http.NewRequest("OPTIONS", "/devices", nil)
				Expect(err).NotTo(HaveOccurred())

				responseRecorder := httptest.NewRecorder()
				goji.DefaultMux.ServeHTTP(responseRecorder, request)
				Expect(responseRecorder.Code).To(Equal(200))
				Expect(responseRecorder.Body.String()).To(Equal(""))
				Expect(responseRecorder.Header().Get("Allow")).To(Equal("POST"))
			})
		})
		Describe("/:id", func() {
			Describe("OPTIONS", func() {
				It("Lists all the provided functions.", func() {
					request, err := http.NewRequest("OPTIONS", "/devices/bibitty", nil)
					Expect(err).NotTo(HaveOccurred())

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal(""))
					Expect(responseRecorder.Header().Get("Allow")).To(Equal("DELETE"))
				})
			})
			Describe("DELETE", func() {
				It("will delete a previously added device", func() {
					deviceInfo := server.DeviceDocument{"alakazham"}
					postRequest, data := NewPOSTRequestWithJSON("/devices", deviceInfo)
					goji.DefaultMux.ServeHTTP(httptest.NewRecorder(), postRequest)

					recorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(recorder, NewRequest("DELETE", "/devices/alakazham"))

					Expect(recorder.Code).To(Equal(200))
					Expect(recorder.Body.String()).To(MatchJSON(data))

					deviceCollection := db.C("devices")
					Expect(deviceCollection.Count()).To(Equal(0))
				})
			})
		})
	})
})