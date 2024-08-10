package virtual_test

import (
	"context"
	"github.com/synnaxlabs/x/binary"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx   = context.Background()
	codec = &binary.JSONCodec{}
)

func TestVirtual(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtual Suite")
}
