package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/anachronistic/apns"
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

type ApnsMockClient struct {
	Response          *apns.PushNotificationResponse
	NotificationsSent []*apns.PushNotification
}

func (self *ApnsMockClient) ConnectAndWrite(response *apns.PushNotificationResponse, payload []byte) error {
	return nil
}

// func (self *ApnsMockClient) ConnectAndWrite(response *apns.PushNotificationResponse, payload []byte) error {
// 	return (*apns.MockClient)(self).ConnectAndWrite(response, payload)
// }

// func (self *ApnsMockClient) OnConnectAndWrite(response *apns.PushNotificationResponse, payload []byte, returnErr error) {
// 	self.On("ConnectAndWrite", response, payload).Return(returnErr)
// }

func (self *ApnsMockClient) Send(pushNotification *apns.PushNotification) *apns.PushNotificationResponse {
	self.NotificationsSent = append(self.NotificationsSent, pushNotification)
	return self.Response
}

// func (self *ApnsMockClient) OnSend(pushNotification *apns.PushNotification, thenReturn *apns.PushNotificationResponse) {
// 	self.On("Send", pushNotification).Return(thenReturn)
// }

var _ = Describe("Endpoint", func() {

	Describe("/devices", func() {

		var db *mgo.Database
		apnsClient := &ApnsMockClient{}

		BeforeEach(func() {
			server.SetupRoutes(TestDatabaseName, &server.APNSCommunicator{func() apns.APNSClient {
				return apnsClient
			}})

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
			It("is able to add a new device.", func() {
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

			It("can be called twice and will return a 200 the second time.", func() {
				deviceInfo := server.DeviceDocument{"abracadabra"}

				request, _ := NewPOSTRequestWithJSON("/devices", deviceInfo)
				goji.DefaultMux.ServeHTTP(httptest.NewRecorder(), request)

				request, data := NewPOSTRequestWithJSON("/devices", deviceInfo)
				responseRecorder := httptest.NewRecorder()
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(200))
				Expect(responseRecorder.Body.String()).To(MatchJSON(data))

				deviceCollection := db.C("devices")
				var result []server.DeviceDocument
				deviceCollection.FindId(deviceInfo.DeviceId).All(&result)
				Expect(result).To(Equal([]server.DeviceDocument{deviceInfo}))
			})

			It("is not able to add a new device when device id is missing.", func() {
				responseRecorder := httptest.NewRecorder()

				request, _ := NewPOSTRequestWithJSON("/devices", struct{ Nothing string }{"nothing"})
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(400))
				Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"POST to /devices must be a JSON with a DeviceId property."}`))

				deviceCollection := db.C("devices")
				Expect(deviceCollection.Count()).To(Equal(0))
			})

			It("is not able to add a new device when device id is an array.", func() {
				responseRecorder := httptest.NewRecorder()

				request, _ := NewPOSTRequestWithJSON("/devices", struct{ Nothing []string }{[]string{"nothing"}})
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(400))
				Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"POST to /devices must be a JSON with a DeviceId property."}`))

				deviceCollection := db.C("devices")
				Expect(deviceCollection.Count()).To(Equal(0))
			})

			It("is not able to add a new device when device id is NULL.", func() {
				responseRecorder := httptest.NewRecorder()

				request, _ := NewPOSTRequestWithJSON("/devices", struct{ Nothing interface{} }{nil})
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(400))
				Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"POST to /devices must be a JSON with a DeviceId property."}`))

				deviceCollection := db.C("devices")
				Expect(deviceCollection.Count()).To(Equal(0))
			})

			It("is not able to add a new device when no JSON is sent.", func() {
				responseRecorder := httptest.NewRecorder()

				request, _ := NewPOSTRequestWithJSON("/devices", nil)
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(400))
				Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"POST to /devices must be a JSON with a DeviceId property."}`))

				deviceCollection := db.C("devices")
				Expect(deviceCollection.Count()).To(Equal(0))
			})

			It("is not able to add a new device when body is not JSON.", func() {
				responseRecorder := httptest.NewRecorder()

				request, _ := NewPOSTRequestWithJSON("/devices", "roguishly devilish raw text")
				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(400))
				Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"POST to /devices must be a JSON with a DeviceId property."}`))

				deviceCollection := db.C("devices")
				Expect(deviceCollection.Count()).To(Equal(0))
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

			Describe("/notifications", func() {
				Describe("OPTIONS", func() {
					It("Lists all the provided functions.", func() {
						request, err := http.NewRequest("OPTIONS", "/devices/token/notifications", nil)
						Expect(err).NotTo(HaveOccurred())

						responseRecorder := httptest.NewRecorder()
						goji.DefaultMux.ServeHTTP(responseRecorder, request)
						Expect(responseRecorder.Code).To(Equal(200))
						Expect(responseRecorder.Body.String()).To(Equal(""))
						Expect(responseRecorder.Header().Get("Allow")).To(Equal("POST"))
					})
				})

				Describe("POST", func() {
					Describe("when sent a notification", func() {
						It("and can forward it to Apple it will respond success", func() {
							message := struct{ Alert string }{"Something important!"}
							request, data := NewPOSTRequestWithJSON("/devices/token/notifications", message)

							apnsClient.Response = apns.NewPushNotificationResponse()

							responseRecorder := httptest.NewRecorder()
							goji.DefaultMux.ServeHTTP(responseRecorder, request)
							Expect(responseRecorder.Code).To(Equal(201))
							Expect(responseRecorder.Body.String()).To(MatchJSON(data))
							Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
							Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal("token"))
							expectedPayload := `{"aps" : {"alert":"Something important!", "badge" : 1}}`
							Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
						})

						It("and can not forward it to Apple it will respond error", func() {
							message := struct{ Alert string }{"Something important!"}
							request, _ := NewPOSTRequestWithJSON("/devices/token/notifications", message)

							apnsClient.Response = apns.NewPushNotificationResponse()
							apnsClient.Response.Error = errors.New("Oh no!")

							responseRecorder := httptest.NewRecorder()
							goji.DefaultMux.ServeHTTP(responseRecorder, request)
							expectedError := `{"Error" : "Oh no!"}`
							Expect(responseRecorder.Code).To(Equal(500))
							Expect(responseRecorder.Body.String()).To(MatchJSON(expectedError))
						})
					})
				})
			})
		})
	})
})
