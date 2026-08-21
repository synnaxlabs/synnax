// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Log {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		l := decode(ctx, "testdata/import_v2.json")
		Expect(l.Channels).To(HaveLen(2))
		Expect(l.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(l.Channels[0].Alias).To(Equal("temp"))
		Expect(l.Channels[0].Notation).To(Equal(notation.NotationScientific))
		Expect(l.Channels[0].Color).To(Equal(color.Color{R: 127, G: 29, B: 29, A: 1}))
		Expect(l.TimestampPrecision).To(Equal(int32(1)))
		Expect(l.ReceiptTimestampHidden).To(BeTrue())
	})

	It("Should lift a Console v1 state through the legacy chain", func(
		ctx SpecContext,
	) {
		l := decode(ctx, "testdata/import_v1.json")
		Expect(l.Channels).To(HaveLen(2))
		Expect(l.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(l.Channels[0].Alias).To(Equal("temp"))
		Expect(l.Channels[0].Precision).To(Equal(int32(2)))
		// showX on the wire inverts into hideX on the current shape.
		Expect(l.ChannelNamesHidden).To(BeFalse())
		Expect(l.ReceiptTimestampHidden).To(BeTrue())
	})

	It("Should lift a Console v0 state of bare channel keys", func(ctx SpecContext) {
		l := decode(ctx, "testdata/import_v0_state.json")
		Expect(l.Channels).To(HaveLen(2))
		Expect(l.Channels[0].Channel).To(Equal(channel.Key(4)))
		Expect(l.Channels[1].Channel).To(Equal(channel.Key(5)))
		Expect(l.Channels[0].Notation).To(Equal(notation.NotationStandard))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v2.json").Key).To(Equal(uuid.Nil))
	})

	It("Should take the name from the envelope header", func(ctx SpecContext) {
		env := LoadEnvelope("testdata/import_v0_state.json")
		env.Name = "Renamed"
		Expect(MustSucceed(versions.DecodeImExEnvelope(ctx, env)).Name).
			To(Equal("Renamed"))
	})

	It("Should reject a version newer than Latest", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_bad_version.json"),
		)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("log version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should return a decode error on a malformed body", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_bad_v2.json"),
		)).Error().To(MatchError(ContainSubstring("decode")))
	})

	It("Should reject a body carrying no log structure", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_unrecognized.json"),
		)).Error().To(MatchError(ContainSubstring(
			`file is not a log: no "channels" field`,
		)))
	})
})
