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
				data, err := json.Marshal(deviceInfo)
				Expect(err).NotTo(HaveOccurred())

				request, err := http.NewRequest("POST", "/devices", bytes.NewReader(data))
				Expect(err).NotTo(HaveOccurred())

				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(201))

				Expect(responseRecorder.HeaderMap.Get("Content-Type")).To(Equal("application/json"))
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
	})
})
