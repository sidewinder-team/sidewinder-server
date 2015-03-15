package main

import (
	"os"
	"strings"

	"github.com/anachronistic/apns"
)

func (self *APNSCommunicator) sendPushNotification(deviceToken string, payload *apns.Payload) error {
	pushNotification := apns.NewPushNotification()
	pushNotification.DeviceToken = deviceToken
	pushNotification.AddPayload(payload)
	client := self.MakeClient()
	response := client.Send(pushNotification)
	return response.Error
}

type APNSCommunicator struct {
	MakeClient func() apns.APNSClient
}

func NewAPNSCommunicator() *APNSCommunicator {
	return &APNSCommunicator{makeAppleNotificationServiceClient}
}

func makeAppleNotificationServiceClient() apns.APNSClient {
	certificate := strings.Replace(os.Getenv("APNS_CERTIFICATE"), "\\n", "\n", -1)
	key := strings.Replace(os.Getenv("APNS_KEY"), "\\n", "\n", -1)
	gateway := os.Getenv("PUSH_GATEWAY")
	return apns.BareClient(gateway, certificate, key)
}
