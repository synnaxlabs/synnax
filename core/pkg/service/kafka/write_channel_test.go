// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Write channels", func() {
	var (
		db       *gorp.DB
		svc      *kafka.Service
		channels *channel.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		svc = MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{
			DB:      db,
			Channel: channels,
		}))
		MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statuses,
			Stores: svc.Stores(),
		}))
	})

	create := func(ctx SpecContext, dt telem.DataType) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     "ch_" + uuid.New().String()[:8],
			DataType: dt,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch.Key()
	}

	config := func(chs ...map[string]any) msgpack.EncodedJSON {
		list := make([]any, len(chs))
		for i, ch := range chs {
			list[i] = ch
		}
		return msgpack.EncodedJSON{"channels": list}
	}

	writeChannel := func(
		kind string,
		ch channel.Key,
		entries ...map[string]any,
	) map[string]any {
		wc := map[string]any{"key": "c1", "type": kind, "channel": ch}
		if len(entries) > 0 {
			list := make([]any, len(entries))
			for i, e := range entries {
				list[i] = e
			}
			wc["enum_values"] = list
		}
		if kind == "timestamp" {
			wc["time_format"] = "unix_ms"
		}
		return wc
	}

	// noEntries matches absent or empty enum values.
	noEntries := Satisfy(func(v any) bool {
		l, _ := v.([]any)
		return len(l) == 0
	})

	entry := func(label string, value any) map[string]any {
		return map[string]any{"label": label, "value": value}
	}

	prepare := func(ctx SpecContext, data msgpack.EncodedJSON) (msgpack.EncodedJSON, error) {
		rec, err := svc.Write.Prepare(ctx, db, uuid.New(), data)
		if err != nil {
			return nil, err
		}
		return rec.Data()
	}

	firstChannel := func(cfg msgpack.EncodedJSON) map[string]any {
		GinkgoHelper()
		chs, ok := cfg["channels"].([]any)
		Expect(ok).To(BeTrue())
		Expect(chs).To(HaveLen(1))
		ch, ok := chs[0].(map[string]any)
		Expect(ok).To(BeTrue())
		return ch
	}

	Describe("Prepare", func() {
		DescribeTable("Should accept each kind a channel's data type allows",
			func(ctx SpecContext, dt telem.DataType, kinds ...string) {
				ch := create(ctx, dt)
				for _, kind := range kinds {
					Expect(prepare(ctx, config(writeChannel(kind, ch)))).Error().
						ToNot(HaveOccurred(), kind)
				}
			},
			Entry("float", telem.Float32T, "float", "string", "boolean"),
			Entry("signed integer", telem.Int16T, "integer", "string", "boolean"),
			Entry("unsigned integer", telem.Uint32T, "unsigned", "string", "boolean"),
			Entry("boolean", telem.BooleanT, "boolean", "unsigned", "string"),
			Entry("string", telem.StringT, "string"),
			Entry("timestamp", telem.TimestampT, "timestamp"),
			Entry("json", telem.JSONT, "json"),
			Entry("uuid", telem.UUIDT, "uuid"),
			Entry("bytes", telem.BytesT, "bytes"),
		)

		DescribeTable("Should reject a kind a channel's data type does not allow",
			func(ctx SpecContext, dt telem.DataType, kind string) {
				ch := create(ctx, dt)
				Expect(prepare(ctx, config(writeChannel(kind, ch)))).Error().To(
					SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							`channel "c1": a `+string(
								dt,
							)+` channel cannot be written as `+
								kind,
						)),
					),
				)
			},
			Entry("a float as an integer", telem.Float64T, "integer"),
			Entry("a signed integer as unsigned", telem.Int8T, "unsigned"),
			Entry("an unsigned integer as a float", telem.Uint8T, "float"),
			Entry("a boolean as an integer", telem.BooleanT, "integer"),
			Entry("a string as a boolean", telem.StringT, "boolean"),
			Entry("a timestamp as a string", telem.TimestampT, "string"),
			Entry("json as a string", telem.JSONT, "string"),
			Entry("a uuid as a string", telem.UUIDT, "string"),
			Entry("bytes as a string", telem.BytesT, "string"),
		)

		DescribeTable("Should reject an entry that does not fit the channel",
			func(ctx SpecContext, dt telem.DataType, wc map[string]any, msg string) {
				wc["channel"] = create(ctx, dt)
				Expect(prepare(ctx, config(wc))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("an integer past the channel's width", telem.Int8T,
				writeChannel("integer", 0, entry("BIG", "200")),
				`enum label "BIG": 200 is out of range for int8`),
			Entry("an unsigned value past a boolean", telem.BooleanT,
				writeChannel("unsigned", 0, entry("TWO", "2")),
				`enum label "TWO": 2 is not a boolean`),
			Entry("text that is not a number", telem.Float64T,
				writeChannel("string", 0, entry("ON", "on")),
				`enum label "ON": "on" is not a float64 value`),
			Entry("floats that narrow to one float32", telem.Float32T,
				writeChannel("float", 0, entry("A", 0.1), entry("B", 0.1000000001)),
				`enum labels "A" and "B" have the same float32 value`),
			Entry("a float past float32", telem.Float32T,
				writeChannel("float", 0, entry("BIG", 1e39)),
				`enum label "BIG": 1e+39 overflows float32`),
			Entry("text that names one number twice", telem.Uint8T,
				writeChannel("string", 0, entry("A", "1"), entry("B", "01")),
				`enum labels "A" and "B" have the same uint8 value`),
		)

		It(
			"Should reject a timestamp channel with no time format",
			func(ctx SpecContext) {
				wc := writeChannel("timestamp", create(ctx, telem.TimestampT))
				delete(wc, "time_format")
				Expect(prepare(ctx, config(wc))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring(`channel "c1": unknown time format ""`),
					),
				))
			},
		)

		It("Should narrow a float entry to a float32 channel", func(ctx SpecContext) {
			ch := create(ctx, telem.Float32T)
			wc := firstChannel(MustSucceed(prepare(ctx, config(
				writeChannel("float", ch, entry("TENTH", 0.1)),
			))))
			Expect(wc["enum_values"]).To(HaveExactElements(HaveKeyWithValue(
				"value", BeNumerically("==", float64(float32(0.1))),
			)))
		})

		It("Should skip a disabled channel, channel 0, and a missing channel",
			func(ctx SpecContext) {
				str := create(ctx, telem.StringT)
				disabled := writeChannel("float", str)
				disabled["disabled"] = true
				unset := writeChannel("float", 0)
				unset["key"] = "c2"
				missing := writeChannel("float", 999999)
				missing["key"] = "c3"
				Expect(prepare(ctx, config(disabled, unset, missing))).Error().
					ToNot(HaveOccurred())
			},
		)
	})

	Describe("Validate", func() {
		DescribeTable("Should reject two entries with one stored value",
			func(ctx SpecContext, kind string, a, b any) {
				Expect(prepare(ctx, config(
					writeChannel(kind, 0, entry("A", a), entry("B", b)),
				))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						`channel "c1": enum_values[1]: value is already labeled`,
					)),
				))
			},
			Entry("two NaN floats", "float", "NaN", "NaN"),
			Entry("zero and negative zero", "float", 0.0, math.Copysign(0, -1)),
			Entry("two integers", "integer", "-3", "-3"),
			Entry("two unsigned integers", "unsigned", "3", "3"),
			Entry("two strings", "string", "on", "on"),
			Entry("two booleans", "boolean", true, true),
		)

		It("Should accept two values that share a label", func(ctx SpecContext) {
			Expect(prepare(ctx, config(writeChannel(
				"unsigned", 0, entry("ON", "1"), entry("ON", "2"),
			)))).Error().ToNot(HaveOccurred())
		})
	})

	Describe("Normalize", func() {
		normalize := func(ctx SpecContext, wc map[string]any) map[string]any {
			GinkgoHelper()
			return firstChannel(MustSucceed(
				svc.Write.Normalize(ctx, db, svc.Write.Version(), config(wc)),
			))
		}

		DescribeTable("Should keep a kind the channel's data type allows",
			func(ctx SpecContext, dt telem.DataType, kind string) {
				wc := normalize(ctx, writeChannel(kind, create(ctx, dt)))
				Expect(wc).To(HaveKeyWithValue("type", kind))
			},
			Entry("a float as a boolean", telem.Float64T, "boolean"),
			Entry("a signed integer as a string", telem.Int64T, "string"),
			Entry("a boolean as unsigned", telem.BooleanT, "unsigned"),
		)

		DescribeTable("Should set the kind of the channel's data type",
			func(
				ctx SpecContext,
				dt telem.DataType,
				from map[string]any,
				kind string,
				entries types.GomegaMatcher,
			) {
				from["channel"] = create(ctx, dt)
				wc := normalize(ctx, from)
				Expect(wc).To(HaveKeyWithValue("type", kind))
				Expect(wc["enum_values"]).To(entries)
			},
			Entry("a float with whole entries to an integer", telem.Int32T,
				writeChannel("float", 0,
					entry("ONE", 1.0), entry("HALF", 0.5), entry("FAULT", "NaN"),
				),
				"integer",
				HaveExactElements(SatisfyAll(
					HaveKeyWithValue("label", "ONE"),
					HaveKeyWithValue("value", "1"),
				))),
			Entry("an integer to unsigned, dropping what does not fit", telem.Uint8T,
				writeChannel("integer", 0,
					entry("ONE", "1"), entry("NEG", "-1"), entry("BIG", "300"),
				),
				"unsigned",
				HaveExactElements(SatisfyAll(
					HaveKeyWithValue("label", "ONE"),
					HaveKeyWithValue("value", "1"),
				))),
			Entry("unsigned to a float, dropping a value past 2^53", telem.Float64T,
				writeChannel("unsigned", 0,
					entry("TWO", "2"), entry("HUGE", "9007199254740993"),
				),
				"float",
				HaveExactElements(SatisfyAll(
					HaveKeyWithValue("label", "TWO"),
					HaveKeyWithValue("value", BeNumerically("==", 2)),
				))),
			Entry("a float to a boolean, dropping its entries", telem.BooleanT,
				writeChannel("float", 0, entry("ON", 1.0)), "boolean", noEntries),
			Entry("a float to a string, dropping its entries", telem.StringT,
				writeChannel("float", 0, entry("ON", 1.0)), "string", noEntries),
			Entry("a float to a timestamp", telem.TimestampT,
				writeChannel("float", 0), "timestamp", noEntries),
			Entry("a string to json, dropping its entries", telem.JSONT,
				writeChannel("string", 0, entry("ON", "on")), "json", noEntries),
			Entry("a boolean to a uuid", telem.UUIDT,
				writeChannel("boolean", 0), "uuid", noEntries),
			Entry("a string to bytes", telem.BytesT,
				writeChannel("string", 0), "bytes", noEntries),
		)

		It("Should give a timestamp the ISO 8601 time format", func(ctx SpecContext) {
			wc := normalize(ctx, writeChannel("integer", create(ctx, telem.TimestampT)))
			Expect(wc).To(HaveKeyWithValue("time_format", "iso8601"))
		})

		DescribeTable("Should keep the first entry that fits the channel",
			func(ctx SpecContext, dt telem.DataType, wc map[string]any) {
				wc["channel"] = create(ctx, dt)
				Expect(normalize(ctx, wc)["enum_values"]).To(HaveExactElements(
					HaveKeyWithValue("label", "A"),
				))
			},
			Entry("after narrowing to a float32", telem.Float32T,
				writeChannel("float", 0, entry("A", 0.1), entry("B", 0.1000000001))),
			Entry("past a boolean", telem.BooleanT,
				writeChannel("unsigned", 0, entry("A", "1"), entry("B", "2"))),
			Entry("past the channel's width", telem.Int8T,
				writeChannel("integer", 0, entry("A", "1"), entry("B", "200"))),
		)

		It("Should keep the kind of a channel that does not exist", func(
			ctx SpecContext,
		) {
			wc := normalize(ctx, writeChannel("float", 999999))
			Expect(wc).To(HaveKeyWithValue("type", "float"))
		})
	})
})

var _ = Describe("Stored write configs", func() {
	It(
		"Should lift v0 channels to the kinds their channels allow",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			db := node.DB
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
			statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			channels := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				DB:      db,
				HostKey: node.Identity.HostKey,
				Status:  statuses,
			}))
			create := func(dt telem.DataType) channel.Key {
				ch := channel.Channel{
					Name:     "ch_" + uuid.New().String()[:8],
					DataType: dt,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channels.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				return ch.Key()
			}
			old := v0.WriteConfig{Topic: "samples", Channels: []v0.WriteChannel{
				{
					Key:      "c1",
					Channel:  create(telem.Uint8T),
					JSONType: payload.JSONTypeNumber,
				},
				{
					Key:        "c2",
					Channel:    create(telem.Int16T),
					JSONType:   payload.JSONTypeString,
					EnumValues: []payload.EnumEntry{{Label: "ON", Value: 1}},
				},
				{
					Key:      "c3",
					Channel:  create(telem.StringT),
					JSONType: payload.JSONTypeString,
				},
			}}
			old.Key = uuid.New()
			Expect(gorp.WrapWriter[uuid.UUID, v0.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			svc := MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{
				DB:      db,
				Channel: channels,
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, old.Key))
			Expect(data["topic"]).To(Equal("samples"))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("type", "unsigned"),
				SatisfyAll(
					HaveKeyWithValue("type", "integer"),
					HaveKeyWithValue("enum_values", HaveExactElements(SatisfyAll(
						HaveKeyWithValue("label", "ON"),
						HaveKeyWithValue("value", "1"),
					))),
				),
				HaveKeyWithValue("type", "string"),
			))
		},
	)
})
