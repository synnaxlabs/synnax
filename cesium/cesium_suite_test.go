package cesium_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"os"
	"testing"
)

var (
	ctx    context.Context
	logger *zap.Logger
)

func TestCaesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Caesium Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	logger = zap.NewNop()
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll("./testdata/cesium")).To(Succeed())
	Expect(os.RemoveAll("./testdata/kv")).To(Succeed())
})
