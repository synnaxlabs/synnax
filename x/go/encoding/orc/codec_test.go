// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc_test

import (
	"bytes"
	"context"

	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

type testRecord struct {
	ID   uint32
	Name string
}

func (t *testRecord) EncodeOrc(w *orc.Writer) error {
	w.Uint32(t.ID)
	w.String(t.Name)
	return nil
}

func (t *testRecord) DecodeOrc(r *orc.Reader) error {
	var err error
	if t.ID, err = r.Uint32(); err != nil {
		return err
	}
	if t.Name, err = r.String(); err != nil {
		return err
	}
	return nil
}

var errEncode = errors.New("encode failed")

type failEncoder struct{}

func (f *failEncoder) EncodeOrc(w *orc.Writer) error { return errEncode }

type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

var _ = Describe("Codec", func() {
	ctx := context.Background()

	Describe("Encode", func() {
		It("Should prepend the magic header", func() {
			data := MustSucceed(orc.Codec.Encode(ctx, &testRecord{ID: 1, Name: "a"}))
			Expect(data[:3]).To(Equal(orc.Magic[:]))
		})

		It("Should propagate encoder errors", func() {
			_, err := orc.Codec.Encode(ctx, &failEncoder{})
			Expect(err).To(MatchError(errEncode))
		})
	})

	Describe("Decode", func() {
		It("Should reject empty data", func() {
			Expect(orc.Codec.Decode(ctx, []byte{}, &testRecord{})).
				To(MatchError(ContainSubstring("invalid magic header")))
		})

		It("Should reject data shorter than 3 bytes", func() {
			Expect(orc.Codec.Decode(ctx, []byte{0x4F, 0x52}, &testRecord{})).
				To(MatchError(ContainSubstring("invalid magic header")))
		})

		It("Should reject wrong magic bytes", func() {
			Expect(orc.Codec.Decode(ctx, []byte{0x00, 0x00, 0x00, 0x00}, &testRecord{})).
				To(MatchError(ContainSubstring("invalid magic header")))
		})
	})

	Describe("Encode/Decode round-trip", func() {
		It("Should round-trip a record", func() {
			in := &testRecord{ID: 42, Name: "hello"}
			data := MustSucceed(orc.Codec.Encode(ctx, in))
			out := &testRecord{}
			Expect(orc.Codec.Decode(ctx, data, out)).To(Succeed())
			Expect(out.ID).To(Equal(uint32(42)))
			Expect(out.Name).To(Equal("hello"))
		})

		It("Should round-trip an empty string", func() {
			in := &testRecord{ID: 0, Name: ""}
			data := MustSucceed(orc.Codec.Encode(ctx, in))
			out := &testRecord{}
			Expect(orc.Codec.Decode(ctx, data, out)).To(Succeed())
			Expect(out.ID).To(Equal(uint32(0)))
			Expect(out.Name).To(Equal(""))
		})
	})

	Describe("EncodeStream/DecodeStream round-trip", func() {
		It("Should round-trip through a buffer", func() {
			in := &testRecord{ID: 99, Name: "stream"}
			var buf bytes.Buffer
			Expect(orc.Codec.EncodeStream(ctx, &buf, in)).To(Succeed())
			out := &testRecord{}
			Expect(orc.Codec.DecodeStream(ctx, &buf, out)).To(Succeed())
			Expect(out.ID).To(Equal(uint32(99)))
			Expect(out.Name).To(Equal("stream"))
		})

		It("Should propagate EncodeStream encoder errors", func() {
			var buf bytes.Buffer
			Expect(orc.Codec.EncodeStream(ctx, &buf, &failEncoder{})).
				To(MatchError(errEncode))
		})

		It("Should propagate DecodeStream read errors", func() {
			readErr := errors.New("read broken")
			Expect(orc.Codec.DecodeStream(ctx, &errReader{err: readErr}, &testRecord{})).
				To(MatchError(readErr))
		})
	})

	Describe("Pool reuse", func() {
		It("Should produce correct results across multiple encode/decode cycles", func() {
			for i := range 10 {
				in := &testRecord{ID: uint32(i), Name: "iter"}
				data := MustSucceed(orc.Codec.Encode(ctx, in))
				out := &testRecord{}
				Expect(orc.Codec.Decode(ctx, data, out)).To(Succeed())
				Expect(out.ID).To(Equal(uint32(i)))
				Expect(out.Name).To(Equal("iter"))
			}
		})
	})
})
