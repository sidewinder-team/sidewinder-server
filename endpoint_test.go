package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
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
	var data []byte
	if value, ok := body.(string); ok {
		data = []byte(value)
	} else {
		jsonData, err := json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())
		data = jsonData
	}

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

func (self *ApnsMockClient) Send(pushNotification *apns.PushNotification) *apns.PushNotificationResponse {
	self.NotificationsSent = append(self.NotificationsSent, pushNotification)
	return self.Response
}

type MockApiCommunicator struct {
	GetUrls     []string
	ResponseMap map[string]*struct {
		Response *http.Response
		Err      error
	}
}

func NewMockApiCommunicator() *MockApiCommunicator {
	communicator := &MockApiCommunicator{}
	communicator.ResponseMap = make(map[string]*struct {
		Response *http.Response
		Err      error
	})
	communicator.ResponseMap[""] = &struct {
		Response *http.Response
		Err      error
	}{}
	return communicator
}

func (self *MockApiCommunicator) SetResponse(url string, status int, body string) {
	getResponse := struct {
		Response *http.Response
		Err      error
	}{}

	getResponse.Response = &http.Response{
		StatusCode: status,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
	}

	if len(body) > 0 {
		buf := bytes.NewBuffer([]byte(body))
		getResponse.Response.Body = ioutil.NopCloser(buf)
	}

	self.ResponseMap[url] = &getResponse
}

func (self *MockApiCommunicator) Get(url string) (*http.Response, error) {
	self.GetUrls = append(self.GetUrls, url)
	getResponse := self.ResponseMap[url]

	if getResponse == nil {
		getResponse = self.ResponseMap[""]
	}

	return getResponse.Response, getResponse.Err
}

var _ = Describe("Endpoint", func() {
	var db *mgo.Database
	var apnsClient *ApnsMockClient
	var apiCommunicator *MockApiCommunicator

	BeforeEach(func() {
		apnsClient = &ApnsMockClient{}
		apnsCommunicator := &server.APNSCommunicator{func() apns.APNSClient {
			return apnsClient
		}}
		apiCommunicator = NewMockApiCommunicator()
		server.SetupRoutes(TestDatabaseName, apnsCommunicator, apiCommunicator)

		session, err := mgo.Dial("mongo,localhost")
		Expect(err).NotTo(HaveOccurred())
		db = session.DB(TestDatabaseName)
		Expect(db).NotTo(BeNil())

		db.C("devices").DropCollection()
		db.C("repositories").DropCollection()
	})

	AfterEach(func() {
		goji.DefaultMux = web.New()
	})

	post := func(path string, data interface{}) {
		request, _ := NewPOSTRequestWithJSON(path, data)
		goji.DefaultMux.ServeHTTP(httptest.NewRecorder(), request)
	}

	Describe("/devices", func() {
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

			Describe("/repositories", func() {
				deviceId := "repositoryTestDevice"
				BeforeEach(func() {
					deviceInfo := server.DeviceDocument{deviceId}
					request, _ := NewPOSTRequestWithJSON("/devices", deviceInfo)
					goji.DefaultMux.ServeHTTP(httptest.NewRecorder(), request)
				})

				Describe("GET", func() {
					It("will start by returning an empty array.", func() {
						request, err := http.NewRequest("GET", "/devices/"+deviceId+"/repositories", nil)
						Expect(err).NotTo(HaveOccurred())

						responseRecorder := httptest.NewRecorder()
						goji.DefaultMux.ServeHTTP(responseRecorder, request)
						Expect(responseRecorder.Code).To(Equal(200))
						Expect(responseRecorder.Body.String()).To(MatchJSON(`[]`))
					})

					It("will show all repositories posted to this device", func() {
						repositoryName1 := "billandted/excellentadventure"
						post("/devices/"+deviceId+"/repositories", struct{ Name string }{repositoryName1})
						repositoryName2 := "billandted/bogusjourney"
						post("/devices/"+deviceId+"/repositories", struct{ Name string }{repositoryName2})
						post("/devices/differentDevice/repositories", struct{ Name string }{"red herring"})

						request, err := http.NewRequest("GET", "/devices/"+deviceId+"/repositories", nil)
						Expect(err).NotTo(HaveOccurred())

						responseRecorder := httptest.NewRecorder()
						goji.DefaultMux.ServeHTTP(responseRecorder, request)
						Expect(responseRecorder.Code).To(Equal(200))
						Expect(responseRecorder.Body.String()).To(MatchJSON(`[{"Name":"` + repositoryName1 + `"},{"Name":"` + repositoryName2 + `"}]`))
					})
				})
				Describe("POST", func() {
					It("will return 201 and the value when successfully put", func() {
						repositoryName := "billandted/excellentadventure"

						request, _ := NewPOSTRequestWithJSON("/devices/"+deviceId+"/repositories",
							struct{ Name string }{repositoryName})

						responseRecorder := httptest.NewRecorder()
						goji.DefaultMux.ServeHTTP(responseRecorder, request)
						Expect(responseRecorder.Code).To(Equal(201))
						Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Name":"` + repositoryName + `"}`))
					})

					It("will return 200 when value is already there", func() {
						repositoryName := "billandted/excellentadventure"

						post("/devices/"+deviceId+"/repositories", struct{ Name string }{repositoryName})
						request, _ := NewPOSTRequestWithJSON("/devices/"+deviceId+"/repositories",
							struct{ Name string }{repositoryName})

						responseRecorder := httptest.NewRecorder()
						goji.DefaultMux.ServeHTTP(responseRecorder, request)
						Expect(responseRecorder.Code).To(Equal(200))
						Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Name":"` + repositoryName + `"}`))
					})
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
							expectedPayload := `{"aps" : {"alert":"Something important!", "badge" : -1}}`
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

	Describe("/hooks", func() {
		Describe("/github", func() {
			deviceId := "MotherBox"
			repositoryName := "apokalypse/anti-life"

			Describe("when the device is registered on that repository", func() {
				BeforeEach(func() {
					post("/devices/"+deviceId+"/repositories", struct{ Name string }{repositoryName})
				})

				It("when this state is failure will always notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("", 200, `[{"state":"success"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"failure","description":"Fun!","branches":[{"Name":"master"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
					expectedPayload := `{"aps" : {"alert":"apokalypse/anti-life: Fun!", "badge" : -1}}`
					Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
					Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal(deviceId))
				})

				It("when this state is error will always notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("", 200, `[{"state":"success"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"error","description":"Fun!","branches":[{"Name":"master"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
					expectedPayload := `{"aps" : {"alert":"apokalypse/anti-life: Fun!", "badge" : -1}}`
					Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
					Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal(deviceId))
				})

				It("when recieving a success and there was a recent failure will notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("https://api.github.com/repos/apokalypse/anti-life/commits/master^/statuses", 200, `[{"state":"failure"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"master"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
					expectedPayload := `{"aps" : {"alert":"apokalypse/anti-life: Fun!", "badge" : -1}}`
					Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
					Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal(deviceId))
				})

				It("when recieving anything but success and there was a recent failure will not notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("https://api.github.com/repos/apokalypse/anti-life/commits/master^/statuses", 200, `[{"state":"failure"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"intermediate","description":"Fun!","branches":[{"Name":"master"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(0))
				})

				It("when there was a recent failure on specific branch will notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse(
						"https://api.github.com/repos/apokalypse/anti-life/commits/experiment^/statuses",
						200, `[{"state":"failure"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"experiment"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
					expectedPayload := `{"aps" : {"alert":"apokalypse/anti-life: Fun!", "badge" : -1}}`
					Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
					Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal(deviceId))
				})

				It("when there was a recent error on specific branch will notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("https://api.github.com/repos/apokalypse/anti-life/commits/experiment^/statuses",
						200, `[{"state":"error"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"experiment"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(1))
					expectedPayload := `{"aps" : {"alert":"apokalypse/anti-life: Fun!", "badge" : -1}}`
					Expect(apnsClient.NotificationsSent[0].PayloadJSON()).To(MatchJSON(expectedPayload))
					Expect(apnsClient.NotificationsSent[0].DeviceToken).To(Equal(deviceId))
				})

				It("when the status does not include a branch return an error.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}
					apiCommunicator.SetResponse("", 200, `[{"state":"failure"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"","description":"Fun!","branches":[]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(400))
					Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"Did not recieve a valid branch in Github status."}`))
				})

				It("when there was a success more recently than the failure will not notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}

					apiCommunicator.SetResponse("https://api.github.com/repos/apokalypse/anti-life/commits/master^/statuses",
						200, `[{"state":"success"},{"state":"failure"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"master"}]}`)

					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(0))
				})

				It("when there not a recent failure will not notify a device of new state.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}

					apiCommunicator.SetResponse("https://api.github.com/repos/apokalypse/anti-life/commits/master^/statuses",
						200, `[{"state":"success"}]`)

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"master"}]}`)
					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(200))
					Expect(responseRecorder.Body.String()).To(Equal("Accepted."))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(0))
				})

				It("when github is not available errors are handled.", func() {
					apnsClient.Response = &apns.PushNotificationResponse{}

					apiCommunicator.ResponseMap[""].Err = errors.New("OH NO")

					request, _ := NewPOSTRequestWithJSON("/hooks/github",
						`{"name":"apokalypse/anti-life","context":"","state":"success","description":"Fun!","branches":[{"Name":"master"}]}`)
					responseRecorder := httptest.NewRecorder()
					goji.DefaultMux.ServeHTTP(responseRecorder, request)
					Expect(responseRecorder.Code).To(Equal(500))
					Expect(responseRecorder.Body.String()).To(MatchJSON(`{"Error":"OH NO"}`))

					Expect(len(apnsClient.NotificationsSent)).To(Equal(0))
					Expect(len(apiCommunicator.GetUrls)).To(Equal(1))
					Expect(apiCommunicator.GetUrls[0]).To(Equal("https://api.github.com/repos/apokalypse/anti-life/commits/master^/statuses"))
				})
			})
		})
	})
})
