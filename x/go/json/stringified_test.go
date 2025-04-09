package json_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/json"
)

var _ = Describe("Stringified", func() {
	Describe("UnmarshalJSON", func() {
		It("Should handle raw JSON strings", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`"hello world"`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal("hello world"))
		})

		It("Should handle JSON objects", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`{"key": "value"}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(`{"key":"value"}`))
		})

		It("Should handle JSON arrays", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`[1, 2, 3]`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(`[1,2,3]`))
		})

		It("Should return error on invalid JSON", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`invalid json`))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("NewStaticString", func() {
		It("Should create a new string from static data", func() {
			data := map[string]interface{}{
				"key": "value",
			}
			s := json.NewStaticString(context.Background(), data)
			Expect(string(s)).To(Equal(`{"key":"value"}`))
		})

		It("Should handle primitive types", func() {
			s := json.NewStaticString(context.Background(), "hello")
			Expect(string(s)).To(Equal(`"hello"`))
		})

		It("Should handle arrays", func() {
			data := []int{1, 2, 3}
			s := json.NewStaticString(context.Background(), data)
			Expect(string(s)).To(Equal(`[1,2,3]`))
		})
	})
})
