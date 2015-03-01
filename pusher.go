package main

import "github.com/anachronistic/apns"

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
	return apns.NewClient("gateway.sandbox.push.apple.com:2195", "certificateFile", "keyFile")
}