package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSidewinderServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SidewinderServer Suite")
}
