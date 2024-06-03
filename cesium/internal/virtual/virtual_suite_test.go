package virtual_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/cesium/internal/testutil"
)

var (
	ctx         = context.Background()
	fileSystems = FileSystems
)

func TestVirtual(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtual Suite")
}
