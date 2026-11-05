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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v0 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Channel", func() {
	Describe("Key", func() {
		It("Should compose the key from the leaseholder and local key", func() {
			ch := v0.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.Key()).To(Equal(channel.NewKey(1, 2)))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the same value as Key", func() {
			ch := v0.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.GorpKey()).To(Equal(ch.Key()))
		})
	})

	Describe("SetOptions", func() {
		It("Should bootstrap-lease a free channel", func() {
			ch := v0.Channel{Leaseholder: node.KeyFree}
			Expect(ch.SetOptions()).To(Equal([]any{node.KeyBootstrapper}))
		})
		It("Should lease a non-free channel to its leaseholder", func() {
			ch := v0.Channel{Leaseholder: 2}
			Expect(ch.SetOptions()).To(Equal([]any{node.Key(2)}))
		})
	})

	Describe("Free", func() {
		It("Should return true for a channel with no leaseholder", func() {
			Expect(v0.Channel{Leaseholder: node.KeyFree}.Free()).To(BeTrue())
		})
		It("Should return false for a leased channel", func() {
			Expect(v0.Channel{Leaseholder: 1}.Free()).To(BeFalse())
		})
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode a channel from msgpack", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":        "test",
				"leaseholder": 3,
			}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Name).To(Equal("test"))
			Expect(c.Leaseholder).To(Equal(node.Key(3)))
		})
		It("Should fall back to node_id when leaseholder is absent", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":    "test",
				"node_id": 7,
			}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Leaseholder).To(Equal(node.Key(7)))
		})
		It("Should decode a channel with legacy operations embedded", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":        "fuel_tc_avg",
				"leaseholder": 1,
				"local_key":   5,
				"operations": []map[string]any{
					{"Type": "avg", "ResetChannel": 0, "Duration": 0},
				},
			}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Operations).To(HaveLen(1))
			Expect(c.Operations[0].Type).To(Equal(v0.OperationTypeAvg))
		})
		It("Should return an error when the msgpack bytes are invalid", func() {
			var c v0.Channel
			Expect(msgpack.Unmarshal([]byte{0xc1}, &c)).To(
				MatchError(ContainSubstring("failed to read raw channel msgpack")),
			)
		})
		It("Should return an error when the value is not a channel", func() {
			data := MustSucceed(msgpack.Marshal("not a channel"))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(
				MatchError(ContainSubstring("failed to decode channel from msgpack")),
			)
		})
		It("Should return an error when the legacy node_id field is malformed", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{"node_id": "abc"}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(
				MatchError(
					ContainSubstring("failed to decode legacy node_id from msgpack"),
				),
			)
		})
	})
})

var _ = Describe("Operation", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode an operation with new lowercase msgpack fields", func() {
			original := v0.Operation{
				Type:         v0.OperationTypeAvg,
				ResetChannel: 42,
				Duration:     5000000000,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Operation
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Type).To(Equal(v0.OperationTypeAvg))
			Expect(decoded.ResetChannel).To(Equal(v0.Key(42)))
			Expect(decoded.Duration).To(Equal(telem.TimeSpan(5000000000)))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Type         string
				ResetChannel uint32
				Duration     int64
			}{
				Type:         "max",
				ResetChannel: 10,
				Duration:     1000000000,
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.Operation
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Type).To(Equal(v0.OperationTypeMax))
			Expect(decoded.ResetChannel).To(Equal(v0.Key(10)))
			Expect(decoded.Duration).To(Equal(telem.TimeSpan(1000000000)))
		})
		It("Should return an error when the msgpack bytes are invalid", func() {
			var o v0.Operation
			Expect(msgpack.Unmarshal([]byte{0xc1}, &o)).To(
				MatchError(ContainSubstring("failed to read raw operation msgpack")),
			)
		})
		It("Should return an error when the value is not an operation", func() {
			data := MustSucceed(msgpack.Marshal("not an operation"))
			var o v0.Operation
			Expect(msgpack.Unmarshal(data, &o)).To(
				MatchError(ContainSubstring("failed to decode operation from msgpack")),
			)
		})
		It("Should return an error when a legacy operation field is malformed", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{"ResetChannel": "abc"}))
			var o v0.Operation
			Expect(msgpack.Unmarshal(data, &o)).To(
				MatchError(
					ContainSubstring("failed to decode legacy operation from msgpack"),
				),
			)
		})
	})
})

var _ = Describe("NormalizeKeys", func() {
	It(
		"Should lift a Channel row stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(xmsgpack.Codec)))
			e := v0.Channel{
				LocalKey:    2,
				Leaseholder: 1,
				Name:        "pt_1",
				DataType:    telem.Float32T,
			}
			legacy := gorptestutil.SetPreV54Row(
				ctx,
				kvDB,
				"Channel",
				e.GorpKey(),
				e,
			)
			table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Channel]{
				DB:         db,
				Migrations: []gorp.Migration{v0.NormalizeKeys},
			}))
			var res v0.Channel
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[v0.Key, v0.Channel](e.GorpKey())).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
			Expect(db.Get(ctx, legacy)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
