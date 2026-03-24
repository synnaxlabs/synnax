// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

func testFrame() frame.Frame {
	return frame.Frame{Frame: telem.MultiFrame(
		[]channel.Key{1, 2},
		[]telem.Series{
			{DataType: telem.Float64T, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			{DataType: telem.Int32T, Data: []byte{9, 10, 11, 12}},
		},
	)}
}

var _ = Describe("Translator", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("WriterRequestTranslator", func() {
		t := framerpb.WriterRequestTranslator{}

		It("Should round-trip a writer request", func() {
			original := writer.Request{
				Command: writer.CommandWrite,
				Config: writer.Config{
					ControlSubject: control.Subject{
						Key:  "test-key",
						Name: "test-name",
					},
					Keys:                     channel.Keys{1, 2, 3},
					Start:                    telem.TimeStamp(1000),
					Authorities:              []control.Authority{200, 100},
					ErrOnUnauthorized:        new(true),
					Mode:                     ts.WriterModePersistStream,
					EnableAutoCommit:         new(false),
					AutoIndexPersistInterval: telem.TimeSpan(5000),
				},
				Frame: testFrame(),
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Command).To(Equal(original.Command))
			Expect(result.Config.ControlSubject.Key).To(Equal("test-key"))
			Expect(result.Config.ControlSubject.Name).To(Equal("test-name"))
			Expect(result.Config.Keys).To(Equal(original.Config.Keys))
			Expect(result.Config.Start).To(Equal(original.Config.Start))
			Expect(result.Config.Authorities).To(Equal(original.Config.Authorities))
			Expect(*result.Config.ErrOnUnauthorized).To(BeTrue())
			Expect(*result.Config.EnableAutoCommit).To(BeFalse())
			Expect(result.Config.Mode).To(Equal(original.Config.Mode))
			Expect(result.Config.AutoIndexPersistInterval).To(
				Equal(original.Config.AutoIndexPersistInterval),
			)
			Expect(result.Frame.Count()).To(Equal(2))
		})

		It("Should handle nil optional fields", func() {
			original := writer.Request{
				Command: writer.CommandOpen,
				Config: writer.Config{
					ControlSubject: control.Subject{Key: "k"},
					Keys:           channel.Keys{1},
				},
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Config.ErrOnUnauthorized).ToNot(BeNil())
			Expect(result.Config.EnableAutoCommit).ToNot(BeNil())
		})
	})

	Describe("WriterResponseTranslator", func() {
		t := framerpb.WriterResponseTranslator{}

		It("Should round-trip a writer response", func() {
			original := writer.Response{
				Command:    writer.CommandCommit,
				SeqNum:     42,
				NodeKey:    5,
				Authorized: true,
				End:        telem.TimeStamp(9999),
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Command).To(Equal(original.Command))
			Expect(result.SeqNum).To(Equal(original.SeqNum))
			Expect(result.NodeKey).To(Equal(original.NodeKey))
			Expect(result.Authorized).To(BeTrue())
			Expect(result.End).To(Equal(original.End))
		})

		It("Should handle zero-value response", func() {
			original := writer.Response{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Command).To(Equal(writer.Command(0)))
			Expect(result.SeqNum).To(Equal(0))
			Expect(result.Authorized).To(BeFalse())
		})
	})

	Describe("IteratorRequestTranslator", func() {
		t := framerpb.IteratorRequestTranslator{}

		It("Should round-trip an iterator request", func() {
			original := iterator.Request{
				Command:   iterator.CommandNext,
				Span:      telem.TimeSpan(5000),
				Bounds:    telem.TimeRange{Start: 100, End: 200},
				Stamp:     telem.TimeStamp(150),
				Keys:      channel.Keys{10, 20},
				ChunkSize: 1024,
				SeqNum:    7,
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Command).To(Equal(original.Command))
			Expect(result.Span).To(Equal(original.Span))
			Expect(result.Bounds).To(Equal(original.Bounds))
			Expect(result.Stamp).To(Equal(original.Stamp))
			Expect(result.Keys).To(Equal(original.Keys))
			Expect(result.ChunkSize).To(Equal(original.ChunkSize))
			Expect(result.SeqNum).To(Equal(original.SeqNum))
		})

		It("Should handle zero-value request", func() {
			original := iterator.Request{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Command).To(Equal(iterator.Command(0)))
			Expect(result.Keys).To(HaveLen(0))
		})
	})

	Describe("IteratorResponseTranslator", func() {
		t := framerpb.IteratorResponseTranslator{}

		It("Should round-trip an iterator response", func() {
			original := iterator.Response{
				Variant: iterator.ResponseVariantData,
				NodeKey: 3,
				Ack:     true,
				SeqNum:  12,
				Command: iterator.CommandNext,
				Frame:   testFrame(),
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Variant).To(Equal(original.Variant))
			Expect(result.NodeKey).To(Equal(original.NodeKey))
			Expect(result.Ack).To(BeTrue())
			Expect(result.SeqNum).To(Equal(original.SeqNum))
			Expect(result.Command).To(Equal(original.Command))
			Expect(result.Frame.Count()).To(Equal(2))
		})

		It("Should handle zero-value response", func() {
			original := iterator.Response{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Ack).To(BeFalse())
			Expect(result.Frame.Empty()).To(BeTrue())
		})
	})

	Describe("RelayRequestTranslator", func() {
		t := framerpb.RelayRequestTranslator{}

		It("Should round-trip a relay request", func() {
			original := relay.Request{Keys: channel.Keys{5, 10, 15}}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Keys).To(Equal(original.Keys))
		})

		It("Should handle empty keys", func() {
			original := relay.Request{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Keys).To(HaveLen(0))
		})
	})

	Describe("RelayResponseTranslator", func() {
		t := framerpb.RelayResponseTranslator{}

		It("Should round-trip a relay response", func() {
			original := relay.Response{
				Frame: testFrame(),
				Group: 42,
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Frame.Count()).To(Equal(2))
			Expect(result.Group).To(Equal(uint32(42)))
		})

		It("Should handle zero-value response", func() {
			original := relay.Response{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Frame.Empty()).To(BeTrue())
			Expect(result.Group).To(Equal(uint32(0)))
		})
	})

	Describe("DeleteRequestTranslator", func() {
		t := framerpb.DeleteRequestTranslator{}

		It("Should round-trip a delete request", func() {
			original := deleter.Request{
				Keys:   channel.Keys{1, 2},
				Names:  []string{"ch1", "ch2"},
				Bounds: telem.TimeRange{Start: 1000, End: 2000},
			}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Keys).To(Equal(original.Keys))
			Expect(result.Names).To(Equal(original.Names))
			Expect(result.Bounds).To(Equal(original.Bounds))
		})

		It("Should handle empty delete request", func() {
			original := deleter.Request{}
			pb := MustSucceed(t.Forward(ctx, original))
			result := MustSucceed(t.Backward(ctx, pb))
			Expect(result.Keys).To(HaveLen(0))
			Expect(result.Names).To(HaveLen(0))
		})
	})
})
