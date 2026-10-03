// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/hex"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v0"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("StatusDetails", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.StatusDetails{Running: true}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should decode legacy uppercase Go field name", func() {
			legacy := struct{ Running bool }{Running: true}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should handle false value correctly for both formats", func() {
			original := v0.StatusDetails{Running: false}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeFalse())
		})
	})
})

var _ = Describe("Mode", func() {
	Describe("IsValid", func() {
		DescribeTable("Should report whether the mode is defined",
			func(m v0.Mode, valid bool) {
				Expect(m.IsValid()).To(Equal(valid))
			},
			Entry("text", v0.ModeText, true),
			Entry("graph", v0.ModeGraph, true),
			Entry("unknown", v0.Mode("bogus"), false),
			Entry("empty", v0.Mode(""), false),
		)
	})
})

var _ = Describe("Arc", func() {
	Describe("GorpKey", func() {
		It("Should return the Arc's key", func() {
			k := uuid.New()
			Expect(v0.Arc{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Arc{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a Arc row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(xmsgpack.Codec)))
			e := v0.Arc{Key: uuid.New(), Name: "Autosequence"}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Arc",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Arc]{
				DB:         db,
				Migrations: []migrate.Migration{v0.NormalizeKeys},
			}))
			var res v0.Arc
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.Arc](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Graph", func() {
	It("Should decode a graph stored by a v0.53 Core", func(ctx SpecContext) {
		// An Arc as a v0.53.4 Core stored it: an "on" node wired to a "write" node.
		stored := MustSucceed(hex.DecodeString(
			"87a47465787482a3415354c0a3726177a0a776657273696f6ea0a46e616d65a96772" +
				"61706820617263a46d6f6465a56772617068a770726f6772616d89a753796d626f6c" +
				"73c0a7547970654d6170c0a946756e6374696f6e73c0a953657175656e636573c0a5" +
				"4e6f646573c0a54564676573c0a6537472617461c0b14f75747075744d656d6f7279" +
				"4261736573c0a45741534dc0a5677261706884a946756e6374696f6e73c0a5456467" +
				"65739183a6536f7572636582a44e6f6465a46e5f6f6ea5506172616da66f75747075" +
				"74a654617267657482a44e6f6465a46e5f7772a5506172616da5696e707574a44b69" +
				"6e6400a54e6f6465739284a6436f6e66696781a76368616e6e656ccb413000090000" +
				"0000a34b6579a46e5f6f6ea454797065a26f6ea8506f736974696f6e82a158cb4024" +
				"000000000000a159cb403400000000000084a6436f6e66696781a76368616e6e656c" +
				"cb4130000a00000000a34b6579a46e5f7772a454797065a57772697465a8506f7369" +
				"74696f6e82a158cb4069000000000000a159cb4034000000000000a856696577706f" +
				"727482a8506f736974696f6e82a158cb0000000000000000a159cb00000000000000" +
				"00a45a6f6f6dca00000000a36b6579c41096e1d792008a4905a3bc5c9fc103cdda",
		))
		var a v0.Arc
		Expect(xmsgpack.Codec.Decode(ctx, stored, &a)).To(Succeed())
		Expect(a.Name).To(Equal("graph arc"))
		Expect(a.Graph.Nodes).To(HaveLen(2))
		Expect(a.Graph.Nodes[0]).To(And(
			HaveField("Key", "n_on"),
			HaveField("Type", "on"),
			HaveField("Config", HaveKeyWithValue("channel", BeEquivalentTo(1048585))),
			HaveField("Position.X", 10.0),
			HaveField("Position.Y", 20.0),
		))
		Expect(a.Graph.Nodes[1]).To(And(
			HaveField("Key", "n_wr"), HaveField("Type", "write"),
		))
		Expect(a.Graph.Edges).To(HaveLen(1))
		Expect(a.Graph.Edges[0]).To(And(
			HaveField("Source.Node", "n_on"),
			HaveField("Source.Param", "output"),
			HaveField("Target.Node", "n_wr"),
			HaveField("Target.Param", "input"),
		))
	})
})
