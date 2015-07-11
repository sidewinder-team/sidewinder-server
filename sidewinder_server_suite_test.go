package main_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSidewinderServer(t *testing.T) {
	RegisterFailHandler(Fail)
	os.MkdirAll("test-results", 0755)
	junitReporter := reporters.NewJUnitReporter("test-results/junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "SidewinderServer Suite", []Reporter{junitReporter})
}
