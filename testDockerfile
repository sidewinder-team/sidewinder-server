FROM golang:1.4

RUN mkdir -p /go/src/github.com/sidewinder-team/sidewinder-server
WORKDIR /go/src/github.com/sidewinder-team/sidewinder-server
COPY . /go/src/github.com/sidewinder-team/sidewinder-server

RUN go-wrapper download \
  && go-wrapper install \
  && go get github.com/onsi/ginkgo/ginkgo \
  && go get github.com/onsi/gomega
CMD ginkgo watch
