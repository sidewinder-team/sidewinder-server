package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anachronistic/apns"
)

func (self *APNSCommunicator) sendPushNotification(deviceToken, alert string) error {
	payload := apns.NewPayload()
	payload.Alert = alert
	payload.Badge = 1

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

	fmt.Printf("CERT:\n%vKEY:\n%v", certificate, key)
	gateway := os.Getenv("PUSH_GATEWAY")
	return apns.BareClient(gateway, certificate, key)
}

// func makeAppleNotificationServiceClient() apns.APNSClient {
// 	return apns.NewClient("gateway.sandbox.push.apple.com:2195", "cert-test.cert", "key-test.key")
// }
