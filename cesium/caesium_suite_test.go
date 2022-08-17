package cesium_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

var ctx = context.Background()

func TestCaesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Caesium Suite")
}

var _ = AfterSuite(func() {
	Expect(os.RemoveAll("./testdata/cesium")).To(Succeed())
	Expect(os.RemoveAll("./testdata/kv")).To(Succeed())
})
