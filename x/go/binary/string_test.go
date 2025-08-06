// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	"bytes"
	"context"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	. "github.com/synnaxlabs/x/testutil"
)

type testStruct struct {
	Name string
	Age  int
}

var _ fmt.Stringer = testStruct{}
var _ binary.StringUnmarshaller = (*testStruct)(nil)

func (t testStruct) String() string {
	return fmt.Sprintf("Name: %s Age: %d", t.Name, t.Age)
}

func (t *testStruct) UnmarshalString(str string) error {
	_, err := fmt.Sscanf(str, "Name: %s Age: %d", &t.Name, &t.Age)
	return err
}

type errorStringUnmarshaller struct{}

var _ binary.StringUnmarshaller = (*errorStringUnmarshaller)(nil)

func (e *errorStringUnmarshaller) UnmarshalString(str string) error {
	return errors.New("error unmarshalling string")
}

var _ = Describe("String", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	DescribeTableSubtree("Encode", func(v any, expected string) {
		It("should encode the records", func() {
			buf := MustSucceed(binary.StringCodec.Encode(ctx, v))
			Expect(string(buf)).To(Equal(expected))
		})
		It("should encode the records to a stream", func() {
			Expect(binary.StringCodec.EncodeStream(ctx, io.Discard, v)).To(Succeed())
		})
	},
		Entry("string", "hello", "hello"),
		Entry("int", 123, "123"),
		Entry("float", 123.456, "123.456"),
		Entry("bool", true, "true"),
		Entry("nil", nil, "<nil>"),
		Entry("struct", struct {
			Name string
			Age  int
		}{Name: "John", Age: 30}, "{John 30}"),
		Entry("fmt.Stringer", testStruct{Name: "John", Age: 30}, "Name: John Age: 30"),
	)
	Describe("with a failing writer", func() {
		It("should return an error", func() {
			Expect(binary.StringCodec.EncodeStream(ctx, xio.AlwaysFailWriter, "hello")).
				To(HaveOccurred())
		})
	})
	DescribeTableSubtree("Decode", func(encoded string, v any, expected any) {
		var p []byte
		BeforeEach(func() {
			p = []byte(encoded)
		})
		It("should decode the records", func() {
			Expect(binary.StringCodec.Decode(ctx, p, v)).To(Succeed())
			Expect(v).To(Equal(expected))
		})
		It("should decode the records from a stream", func() {
			Expect(binary.StringCodec.DecodeStream(ctx, bytes.NewReader(p), v)).To(Succeed())
			Expect(v).To(Equal(expected))
		})
	},
		Entry("fmt.Stringer",
			"Name: John Age: 30",
			&testStruct{},
			&testStruct{Name: "John", Age: 30},
		),
		Entry("*string",
			"hello",
			new(string),
			func() *string { s := "hello"; return &s }(),
		),
	)
	DescribeTableSubtree("Decoding errors", func(encoded string, v any) {
		var p []byte
		BeforeEach(func() {
			p = []byte(encoded)
		})
		It("should return an error", func() {
			Expect(binary.StringCodec.Decode(ctx, p, v)).To(HaveOccurred())
		})
		It("should return an error from a stream", func() {
			Expect(binary.StringCodec.DecodeStream(ctx, bytes.NewReader(p), v)).
				To(HaveOccurred())
		})
	},
		Entry("not a pointer", "hello", ""),
		Entry("integer", "hello", new(int)),
		Entry("doesn't implement StringUnmarshaller", "hello", new(struct{})),
		Entry("error unmarshalling string", "hello", new(errorStringUnmarshaller)),
	)
	Describe("with a failing reader", func() {
		It("should return an error", func() {
			Expect(binary.StringCodec.DecodeStream(ctx, xio.AlwaysFailReader, new(string))).
				To(HaveOccurred())
		})
	})
})
