package main_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/zenazn/goji"

	server "github.com/sidewinder-team/sidewinder-server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Endpoint", func() {
	server.SetupRoutes()

	Describe("/devices", func() {

		Describe("POST", func() {
			It("is able to successfully add a new device.", func() {
				responseRecorder := httptest.NewRecorder()

				deviceInfo := struct {
					DeviceToken string
				}{"abracadabra"}
				data, err := json.Marshal(deviceInfo)
				Expect(err).NotTo(HaveOccurred())

				request, err := http.NewRequest("POST", "/devices", bytes.NewReader(data))
				Expect(err).NotTo(HaveOccurred())

				goji.DefaultMux.ServeHTTP(responseRecorder, request)

				Expect(responseRecorder.Code).To(Equal(201))
				Expect(responseRecorder.HeaderMap.Get("Content-Type")).To(Equal("application/json"))
				Expect(responseRecorder.Body.String()).To(MatchJSON(data))
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
