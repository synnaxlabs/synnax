package date_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Date Suite")
}
