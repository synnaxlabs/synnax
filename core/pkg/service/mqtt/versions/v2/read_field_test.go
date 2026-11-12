// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var base = v2.BaseReadField{Key: "f1", Name: "level", Channel: 7, Pointer: "/v"}

func plainEntry(fields ...v2.ReadField) v2.ReadEntry {
	return v2.ReadEntry{Variant: v2.PlainReadEntry{Fields: fields}}
}

func channelsOf(found map[channel.Key]v2.Channel) v2.Channels {
	return func(
		_ context.Context,
		_ gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v2.Channel, error) {
		out := map[channel.Key]v2.Channel{}
		for _, k := range keys {
			if ch, ok := found[k]; ok {
				out[k] = ch
			}
		}
		return out, nil
	}
}

var _ = Describe("Read field", func() {
	DescribeTable(
		"Should keep the kind and the options through ReadOf and NewReadField",
		func(r payload.Read) {
			f := v2.NewReadField(base, r)
			Expect(f.Type()).To(Equal(r.Kind))
			Expect(f.Base()).To(Equal(base))
			Expect(v2.ReadOf(f)).To(Equal(r))
		},
		Entry("float", payload.FromFloat(payload.FloatRead{
			EnumValues: []payload.FloatEntry{{Label: "HALF", Value: 0.5}},
		})),
		Entry("integer", payload.FromInteger(payload.IntegerRead{
			EnumValues: []payload.IntegerEntry{{Label: "MIN", Value: -1}},
		})),
		Entry("unsigned", payload.FromUnsigned(payload.UnsignedRead{
			EnumValues: []payload.UnsignedEntry{{Label: "ON", Value: 1}},
		})),
		Entry("string", payload.Read{
			Kind:   payload.KindString,
			String: payload.StringRead{Fill: new("none")},
		}),
		Entry("boolean", payload.Read{
			Kind:    payload.KindBoolean,
			Boolean: payload.BooleanRead{Fill: new(true)},
		}),
		Entry("timestamp", payload.Read{
			Kind:      payload.KindTimestamp,
			Timestamp: payload.TimestampRead{TimeFormat: payload.TimeFormatUnixNs},
		}),
		Entry("json", payload.Read{Kind: payload.KindJSON}),
		Entry("uuid", payload.Read{Kind: payload.KindUUID}),
		Entry("bytes", payload.Read{Kind: payload.KindBytes}),
	)

	It("Should panic on an unknown kind", func() {
		Expect(func() { v2.NewReadField(base, payload.Read{Kind: "complex"}) }).
			To(PanicWith("unknown read field kind complex"))
	})

	Describe("PlainFields", func() {
		It("Should visit the fields of each plain entry with their path", func() {
			field := v2.NewReadField(base, payload.Read{Kind: payload.KindJSON})
			cfg := v2.ReadConfig{Entries: []v2.ReadEntry{
				{Variant: v2.SparkplugReadEntry{Tag: "speed"}},
				plainEntry(field),
			}}
			var paths []string
			Expect(v2.PlainFields(&cfg, func(path string, fields []v2.ReadField) error {
				paths = append(paths, path)
				Expect(fields).To(Equal([]v2.ReadField{field}))
				return nil
			})).To(Succeed())
			Expect(paths).To(Equal([]string{"entries.1.fields"}))
		})

		It("Should return the first error of fn", func() {
			cfg := v2.ReadConfig{Entries: []v2.ReadEntry{plainEntry(), plainEntry()}}
			calls := 0
			Expect(v2.PlainFields(&cfg, func(string, []v2.ReadField) error {
				calls++
				return errors.New("stop")
			})).To(MatchError("stop"))
			Expect(calls).To(Equal(1))
		})
	})

	Describe("FieldChannels", func() {
		It("Should return the channels of the fields that have one", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			none := base
			none.Channel = 0
			cfg := v2.ReadConfig{Entries: []v2.ReadEntry{plainEntry(
				v2.NewReadField(base, payload.Read{Kind: payload.KindJSON}),
				v2.NewReadField(none, payload.Read{Kind: payload.KindJSON}),
			)}}
			level := v2.Channel{Name: "level", DataType: telem.JSONT}
			Expect(v2.FieldChannels(ctx, db, channelsOf(map[channel.Key]v2.Channel{
				7: level,
			}), &cfg)).To(Equal(map[channel.Key]v2.Channel{7: level}))
		})

		It("Should read nothing when no field has a channel", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			cfg := v2.ReadConfig{Entries: []v2.ReadEntry{plainEntry()}}
			read := func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]v2.Channel, error) {
				Fail("read channels with no keys")
				return nil, nil
			}
			Expect(v2.FieldChannels(ctx, db, read, &cfg)).To(BeNil())
		})
	})

	Describe("ReconcileReadConfig", func() {
		It("Should change a field to agree with its channel and skip the others", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			missing, none := base, base
			missing.Channel, none.Channel = 8, 0
			cfg := v2.ReadConfig{
				Key: uuid.New(),
				Entries: []v2.ReadEntry{plainEntry(
					v2.NewReadField(base, payload.Read{Kind: payload.KindFloat}),
					v2.NewReadField(missing, payload.Read{Kind: payload.KindFloat}),
					v2.NewReadField(none, payload.Read{Kind: payload.KindFloat}),
				)},
			}
			ins, logs := observe()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return v2.ReconcileReadConfig(
					ctx,
					tx,
					channelsOf(map[channel.Key]v2.Channel{
						7: {Name: "level", DataType: telem.Uint8T},
					}),
					ins,
					&cfg,
				)
			})).To(Succeed())
			plain, ok := cfg.Entries[0].Variant.(v2.PlainReadEntry)
			Expect(ok).To(BeTrue())
			Expect(plain.Fields[0].Type()).To(Equal(payload.KindUnsigned))
			Expect(plain.Fields[1].Type()).To(Equal(payload.KindFloat))
			Expect(plain.Fields[2].Type()).To(Equal(payload.KindFloat))
			Expect(logs.All()).To(HaveLen(1))
			Expect(logs.All()[0].ContextMap()).To(And(
				HaveKeyWithValue("field", "entries.0.fields.0"),
				HaveKeyWithValue("channel_name", "level"),
				HaveKeyWithValue("change", "kind float became unsigned"),
			))
		})
	})
})
