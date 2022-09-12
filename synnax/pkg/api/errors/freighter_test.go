package errors_test

import (
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/arya-analytics/freighter/ferrors"
	roacherrors "github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Freighter", func() {
	Describe("Encode + Decode", func() {
		Context("Validation Err", func() {
			It("Should encode and decode a validation error", func() {
				err := errors.Validation(errors.Fields{
					{
						Field:   "field",
						Message: "message",
					},
					{
						Field:   "field2",
						Message: "message2",
					},
				})
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(Equal(err))
			})
		})
		Context("Message Err", func() {
			It("Should encode and decode a message error", func() {
				err := errors.General(roacherrors.New("my crazy error"))
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(Equal(err))
			})
		})
		Context("Nil Err", func() {
			It("Should encode and decode a nil error", func() {
				err := errors.Nil
				encoded := ferrors.Encode(err)
				decoded := ferrors.Decode(encoded)
				Expect(decoded).To(BeNil())
			})
		})
	})

})
