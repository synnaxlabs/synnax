package resource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/resource"
)

var _ = Describe("Resource", func() {
	Describe("NewErrClosed", func() {
		It("Should return an error with the correct message", func() {
			Expect(resource.NewErrClosed("test")).To(And(
				MatchError(resource.ErrClosed),
				MatchError(ContainSubstring("cannot complete operation on closed test")),
			))
		})
	})

})
