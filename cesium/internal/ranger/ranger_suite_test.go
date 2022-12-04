package ranger_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRanger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ranger Suite")
}
