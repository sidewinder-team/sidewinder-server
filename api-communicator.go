package main

import "net/http"

type ApiCommunicator interface {
	Get(url string) (*http.Response, error)
}

type HttpCommunicator struct{}

func (self HttpCommunicator) Get(url string) (*http.Response, error) {
	return http.Get(url)
}
