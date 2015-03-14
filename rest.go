package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zenazn/goji/web"
)

type RestMux struct {
	Mux     *web.Mux
	pattern string
}

func NewRestMux(pattern string, mux *web.Mux) *RestMux {
	return &RestMux{mux, pattern}
}

func (self *RestMux) Use(endpointHandler RestEndpointHandler) *RestMux {
	endpoint := endpointHandler.Point()
	var methods []string
	if endpoint.Get != nil {
		self.Mux.Get(self.pattern, endpoint.Get)
		methods = append(methods, "GET")
	}
	if endpoint.Post != nil {
		self.Mux.Post(self.pattern, endpoint.Post)
		methods = append(methods, "POST")
	}
	if endpoint.Put != nil {
		self.Mux.Put(self.pattern, endpoint.Put)
		methods = append(methods, "PUT")
	}
	if endpoint.Delete != nil {
		methods = append(methods, "DELETE")
		self.Mux.Delete(self.pattern, endpoint.Delete)
	}
	if len(methods) > 0 {
		self.Mux.Options(self.pattern, func(context web.C, writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Allow", strings.Join(methods, ","))
		})
	}

	for pattern, endpoint := range endpoint.Paths {
		self.Handle(pattern, &endpoint)
	}
	return self
}

func (self *RestMux) Handle(pattern string, endpointHandler RestEndpointHandler) *RestMux {
	endpoint := endpointHandler.Point()
	newRestMux := NewRestMux(self.pattern+pattern, self.Mux)
	newRestMux.Use(endpoint)
	return newRestMux
}

type RestEndpointHandler interface {
	Point() *RestEndpoint
}

type RestEndpointFunc func() *RestEndpoint

func (self RestEndpointFunc) Point() *RestEndpoint {
	return self()
}

type RestEndpoint struct {
	Get    web.Handler
	Put    web.Handler
	Post   web.Handler
	Delete web.Handler
	Paths  map[string]RestEndpoint
}

func (self *RestEndpoint) Point() *RestEndpoint {
	return self
}

func (self *RestEndpoint) Route(pattern string, endpoint RestEndpoint) *RestEndpoint {
	if self.Paths == nil {
		self.Paths = make(map[string]RestEndpoint)
	}
	self.Paths[pattern] = endpoint
	return self
}

type RestHandler func(context web.C, writer http.ResponseWriter, request *http.Request) error

func (self RestHandler) ServeHTTPC(context web.C, writer http.ResponseWriter, request *http.Request) {
	if err := self(context, writer, request); err != nil {
		fmt.Printf("ERROR:  %v\n", err.Error())
		writeJson(500, ErrorJson{err.Error()}, writer)
	}
}

func writeJson(code int, value interface{}, writer http.ResponseWriter) error {
	writer.WriteHeader(code)
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(value)
}
