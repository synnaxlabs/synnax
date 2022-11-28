package cesium_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"os"
	"testing"
)

var (
	ctx    context.Context
	logger *zap.Logger
)

func openMemDB() cesium.DB {
	//logger = lo.Must(zap.NewDevelopment())
	db := MustSucceed(cesium.Open("./testdata"))//cesium.MemBacked(),
	//cesium.WithLogger(logger),

	return db
}

func TestCaesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Caesium Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	logger = MustSucceed(zap.NewDevelopment())
	logger = zap.NewNop()
	zap.ReplaceGlobals(logger)
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll("./testdata/cesium")).To(Succeed())
	Expect(os.RemoveAll("./testdata/kv")).To(Succeed())
})
