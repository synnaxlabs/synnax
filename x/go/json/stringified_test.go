// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

		It("should handle numbers", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`123`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(`123`))
		})

		It("should handle booleans", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`true`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(`true`))
		})

		It("should handle null", func() {
			var s json.String
			err := s.UnmarshalJSON([]byte(`null`))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(s)).To(Equal(``))
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

		It("should handle null", func() {
			s := json.NewStaticString(context.Background(), nil)
			Expect(string(s)).To(Equal(`null`))
		})

		It("should handle numbers", func() {
			s := json.NewStaticString(context.Background(), 1)
			Expect(string(s)).To(Equal(`1`))
		})

		It("should handle booleans", func() {
			s := json.NewStaticString(context.Background(), true)
			Expect(string(s)).To(Equal(`true`))
		})

		It("Should handle arrays", func() {
			data := []int{1, 2, 3}
			s := json.NewStaticString(context.Background(), data)
			Expect(string(s)).To(Equal(`[1,2,3]`))
		})
	})
})
