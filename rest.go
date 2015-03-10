package main

import (
	"encoding/json"
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

func (self *RestMux) Use(handler *RestHandler) *RestMux {
	var methods []string
	if handler.Get != nil {
		self.Mux.Get(self.pattern, handler.Get)
		methods = append(methods, "GET")
	}
	if handler.Post != nil {
		self.Mux.Post(self.pattern, handler.Post)
		methods = append(methods, "POST")
	}
	if handler.Put != nil {
		self.Mux.Put(self.pattern, handler.Put)
		methods = append(methods, "PUT")
	}
	if handler.Delete != nil {
		methods = append(methods, "DELETE")
		self.Mux.Delete(self.pattern, handler.Delete)
	}
	if len(methods) > 0 {
		self.Mux.Options(self.pattern, func(context web.C, writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Allow", strings.Join(methods, ","))
		})
	}
	return self
}

func (self *RestMux) Handle(pattern string, handler *RestHandler) *RestMux {
	newRestMux := NewRestMux(self.pattern+pattern, self.Mux)
	newRestMux.Use(handler)
	return newRestMux
}

type RestHandler struct {
	Get    web.HandlerType
	Put    web.HandlerType
	Post   web.HandlerType
	Delete web.HandlerType
}

type Handler func(context web.C, writer http.ResponseWriter, request *http.Request) error

func catchErr(handler Handler) web.HandlerFunc {
	return web.HandlerFunc(func(context web.C, writer http.ResponseWriter, request *http.Request) {
		if err := handler(context, writer, request); err != nil {
			writeJson(500, ErrorJson{err.Error()}, writer)
		}
	})
}

func writeJson(code int, value interface{}, writer http.ResponseWriter) error {
	writer.WriteHeader(code)
	writer.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(writer).Encode(value)
}
