package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"os"
	"testing"
)

var logger *zap.Logger

func openMemDB() cesium.DB {
	return MustSucceed(cesium.Open(
		"./testdata",
		cesium.MemBacked(),
		cesium.WithLogger(logger),
	))
}

func TestCaesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Caesium Suite")
}

var _ = BeforeSuite(func() {
	logger = MustSucceed(zap.NewDevelopment())
	logger = zap.NewNop()
	zap.ReplaceGlobals(logger)
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll("./testdata/cesium")).To(Succeed())
	Expect(os.RemoveAll("./testdata/kv")).To(Succeed())
})
