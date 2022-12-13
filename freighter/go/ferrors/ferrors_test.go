package ferrors_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/ferrors"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	MyCustomErrorType     ferrors.Type = "my-custom-error"
	UnregisteredErrorType ferrors.Type = "unregistered-error"
)

var (
	MyCustomErrorOne = ferrors.Typed(errors.New("one"), MyCustomErrorType)
	MyCustomErrorTwo = ferrors.Typed(errors.New("two"), MyCustomErrorType)
	MyCustomNil      = ferrors.Typed(errors.New("nil"), MyCustomErrorType)
)

func encodeMyCustomError(err error) string {
	return err.Error()
}

func decodeMyCustomError(encoded string) error {
	switch encoded {
	case MyCustomErrorOne.Error():
		return MyCustomErrorOne
	case MyCustomErrorTwo.Error():
		return MyCustomErrorTwo
	case MyCustomNil.Error():
		return MyCustomNil
	}
	panic("unknown error")
}

var _ = Describe("Ferrors", func() {
	Describe("New", func() {
		It("Should register a custom error type correctly", func() {
			Expect(func() {
				ferrors.Register(MyCustomErrorType, encodeMyCustomError, decodeMyCustomError)
			}).ToNot(Panic())
		})
		It("Should panic if the error type is already registered", func() {
			Expect(func() {
				ferrors.Register(MyCustomErrorType, encodeMyCustomError, decodeMyCustomError)
			}).To(Panic())
		})
	})
	Describe("Encode", func() {
		It("Should encode a custom error type into a payload", func() {
			pld := ferrors.Encode(MyCustomErrorOne)
			Expect(pld.Type).To(Equal(MyCustomErrorType))
			Expect(pld.Data).To(Equal(MyCustomErrorOne.Error()))
		})
		It("Should encode a nil error into a Nil typed payload", func() {
			pld := ferrors.Encode(nil)
			Expect(pld.Type).To(Equal(ferrors.Nil))
			Expect(pld.Data).To(BeEmpty())
		})
		It("Should encode a custom nil type into a Nil typed payload", func() {
			pld := ferrors.Encode(MyCustomNil)
			Expect(pld.Type).To(Equal(ferrors.Nil))
			Expect(pld.Data).To(BeEmpty())
		})
		It("Should encode an unknown error using cockroachdb's errors package", func() {
			pld := ferrors.Encode(errors.New("unknown"))
			Expect(pld.Type).To(Equal(ferrors.Roach))
		})
		It("Should return an unknown error type if the error type is not registered", func() {
			pld := ferrors.Encode(ferrors.Typed(errors.New("unknown"), UnregisteredErrorType))
			Expect(pld.Type).To(Equal(ferrors.Unknown))
		})
	})
	Describe("Decode", func() {
		It("Should decode a custom error type from a payload", func() {
			pld := ferrors.Encode(MyCustomErrorOne)
			err := ferrors.Decode(pld)
			Expect(err).To(Equal(MyCustomErrorOne))
		})
		It("Should decode a nil error from a Nil typed payload", func() {
			pld := ferrors.Encode(nil)
			err := ferrors.Decode(pld)
			Expect(err).To(BeNil())
		})
		It("Should decode an unknown error using cockroachdb's errors package", func() {
			pld := ferrors.Encode(errors.New("unknown"))
			err := ferrors.Decode(pld)
			Expect(err).To(HaveOccurredAs(errors.New("unknown")))
		})
	})
})
