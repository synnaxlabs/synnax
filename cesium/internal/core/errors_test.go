package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/cesium/internal/core"
)

var _ = Describe("Errors", func() {
	Describe("NewErrChannelNotFound", func() {
		It("Should return an error with the correct message", func() {
			err := core.NewErrChannelNotFound(1)
			Expect(err).To(HaveOccurredAs(core.ErrChannelNotFound))
			Expect(err).To(MatchError(ContainSubstring("channel 1 not found")))
		})
	})

	Describe("NewErrEntityClosed", func() {
		It("Should return an error with the correct message", func() {
			err := core.NewErrEntityClosed("writer")
			Expect(err).To(HaveOccurredAs(core.ErrClosedEntity))
			Expect(err).To(MatchError(ContainSubstring("cannot complete operation on closed writer")))
		})
	})

	Describe("NewChannelErrWrapper", func() {
		It("Should return an error with the correct message", func() {
			ch := core.Channel{Key: 1, Name: "foo"}
			err := core.NewChannelErrWrapper(ch)(errors.Newf("bad error"))
			Expect(err).To(MatchError(ContainSubstring("channel [foo]<1>: bad error")))
		})
		It("Should return nil if the error is nil", func() {
			ch := core.Channel{Key: 1, Name: "foo"}
			err := core.NewChannelErrWrapper(ch)(nil)
			Expect(err).To(BeNil())
		})
	})
})
