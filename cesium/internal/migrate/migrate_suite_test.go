package migrate_test

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx         = context.Background()
	fileSystems = testutil.FileSystems
)

func TestMigrate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migrate Suite")
}
