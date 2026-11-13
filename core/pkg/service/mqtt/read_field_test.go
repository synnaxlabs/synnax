// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"context"
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Read field", func() {
	var (
		db       *gorp.DB
		svc      *mqtt.Service
		channels *channel.Service
		node     testutil.Node
	)
	BeforeEach(func(ctx SpecContext) {
		node = testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		svc = MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
			DB:      db,
			Channel: channels,
		}))
	})

	createChannel := func(ctx context.Context, dt telem.DataType) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: UniqueChannelName(), DataType: dt, Virtual: true}
		if dt == telem.TimestampT {
			ch = channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    dt,
				IsIndex:     true,
				Leaseholder: node.Identity.HostKey,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	readConfig := func(fields ...map[string]any) msgpack.EncodedJSON {
		list := make([]any, len(fields))
		for i, f := range fields {
			list[i] = f
		}
		return msgpack.EncodedJSON{"entries": []any{map[string]any{
			"key":    "entry",
			"type":   "plain",
			"topic":  "plant/x",
			"fields": list,
		}}}
	}

	readField := func(
		kind string,
		ch channel.Channel,
		options map[string]any,
	) map[string]any {
		f := map[string]any{
			"type":    kind,
			"key":     UniqueChannelName(),
			"channel": ch.Key(),
			"pointer": "/v",
		}
		maps.Copy(f, options)
		return f
	}

	fieldsOf := func(data msgpack.EncodedJSON) []any {
		GinkgoHelper()
		entries := data["entries"].([]any)
		return entries[0].(map[string]any)["fields"].([]any)
	}

	Describe("Write", func() {
		prepare := func(
			ctx context.Context,
			cfg msgpack.EncodedJSON,
		) (msgpack.EncodedJSON, error) {
			rec, err := svc.Read.Prepare(ctx, db, uuid.New(), "", cfg)
			if err != nil {
				return nil, err
			}
			return rec.Data()
		}

		It("Should clear the data type of a field with a channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Int16T)
			data := MustSucceed(prepare(ctx, readConfig(
				readField("integer", ch, map[string]any{"data_type": "int16"}),
			)))
			Expect(fieldsOf(data)[0]).ToNot(HaveKey("data_type"))
		})

		It("Should keep the data type of a field with no channel", func(
			ctx SpecContext,
		) {
			data := MustSucceed(prepare(ctx, readConfig(readField(
				"unsigned", channel.Channel{}, map[string]any{"data_type": "uint8"},
			))))
			Expect(fieldsOf(data)[0]).To(HaveKeyWithValue("data_type", "uint8"))
		})

		It("Should skip a disabled field and a channel that does not exist", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Int16T)
			disabled := readField("float", ch, map[string]any{"disabled": true})
			missing := readField("float", ch, nil)
			missing["channel"] = 4000000
			MustSucceed(prepare(ctx, readConfig(disabled, missing)))
		})

		It("Should skip the fields of a disabled entry", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Int32T)
			cfg := readConfig(readField("float", ch, nil))
			cfg["entries"].([]any)[0].(map[string]any)["disabled"] = true
			data := MustSucceed(prepare(ctx, cfg))
			Expect(fieldsOf(data)[0]).To(HaveKeyWithValue("type", "float"))
		})

		DescribeTable(
			"should reject a field that does not agree with its channel",
			func(
				ctx SpecContext,
				dt telem.DataType,
				kind string,
				options map[string]any,
				message string,
			) {
				ch := createChannel(ctx, dt)
				Expect(prepare(ctx, readConfig(readField(kind, ch, options)))).
					Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry(
				"a kind that the channel does not take",
				telem.Uint8T,
				"integer",
				nil,
				"a field of kind integer cannot read into a channel of data type uint8",
			),
			Entry("an entry past the width",
				telem.Int8T, "integer",
				map[string]any{"enum_values": []any{
					map[string]any{"label": "BIG", "value": "200"},
				}},
				`enum label "BIG": 200 does not fit int8`),
			Entry("a fill past the width",
				telem.Float32T, "float", map[string]any{"fill": 1e39},
				"fill: 1e+39 does not fit float32"),
			Entry("a fill on an index channel",
				telem.TimestampT, "timestamp",
				map[string]any{"time_format": "unix_ms", "fill": "0"},
				"an index channel cannot take a fill"),
		)

		DescribeTable(
			"should reject a field that is not valid",
			func(ctx SpecContext, fields []map[string]any, message string) {
				Expect(prepare(ctx, readConfig(fields...))).Error().
					To(MatchError(ContainSubstring(message)))
			},
			Entry("a repeated enum label",
				[]map[string]any{readField(
					"string", channel.Channel{}, map[string]any{"enum_values": []any{
						map[string]any{"label": "A", "value": "x"},
						map[string]any{"label": "A", "value": "y"},
					}},
				)},
				`enum label "A" is repeated`),
			Entry("a timestamp with no time format",
				[]map[string]any{readField("timestamp", channel.Channel{}, nil)},
				"invalid time_format"),
			Entry("a raw bytes field beside another field",
				[]map[string]any{
					readField(
						"bytes",
						channel.Channel{},
						map[string]any{"pointer": ""},
					),
					readField("float", channel.Channel{}, nil),
				},
				"a bytes field with an empty pointer must be the only enabled field"),
		)
	})

	Describe("Normalize", func() {
		normalize := func(
			ctx context.Context,
			fields ...map[string]any,
		) map[string]any {
			GinkgoHelper()
			data := MustSucceed(svc.Read.Normalize(ctx, db, 2, readConfig(fields...)))
			return fieldsOf(data)[0].(map[string]any)
		}

		DescribeTable(
			"should set the kind that the channel takes",
			func(ctx SpecContext, dt telem.DataType, kind string) {
				ch := createChannel(ctx, dt)
				Expect(normalize(ctx, readField("string", ch, nil))).
					To(HaveKeyWithValue("type", kind))
			},
			Entry("float", telem.Float32T, "float"),
			Entry("integer", telem.Int64T, "integer"),
			Entry("unsigned", telem.Uint16T, "unsigned"),
			Entry("boolean", telem.BooleanT, "boolean"),
			Entry("timestamp", telem.TimestampT, "timestamp"),
			Entry("json", telem.JSONT, "json"),
			Entry("uuid", telem.UUIDT, "uuid"),
			Entry("bytes", telem.BytesT, "bytes"),
		)

		It("Should keep a field whose channel takes its kind", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.StringT)
			Expect(normalize(ctx, readField("float", ch, nil))).
				To(HaveKeyWithValue("type", "string"))
			Expect(normalize(ctx, readField("string", ch, map[string]any{
				"enum_values": []any{map[string]any{"label": "A", "value": "x"}},
			}))).To(SatisfyAll(
				HaveKeyWithValue("type", "string"),
				HaveKeyWithValue("enum_values", HaveLen(1)),
			))
		})

		It("Should convert the entries and fill that the new kind holds", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Uint8T)
			Expect(normalize(ctx, readField("float", ch, map[string]any{
				"data_type": "float64",
				"fill":      7,
				"enum_values": []any{
					map[string]any{"label": "ON", "value": 1},
					map[string]any{"label": "HALF", "value": 0.5},
					map[string]any{"label": "BIG", "value": 256},
					map[string]any{"label": "NEG", "value": -1},
					map[string]any{"label": "NAN", "value": "NaN"},
				},
			}))).To(SatisfyAll(
				HaveKeyWithValue("type", "unsigned"),
				Not(HaveKey("data_type")),
				HaveKeyWithValue("fill", "7"),
				HaveKeyWithValue("enum_values", HaveExactElements(
					map[string]any{"label": "ON", "value": "1"},
				)),
			))
		})

		It("Should drop a non-finite fill on an integer kind", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Int32T)
			Expect(normalize(ctx, readField("float", ch, map[string]any{
				"fill": "NaN",
			}))).ToNot(HaveKey("fill"))
		})

		It("Should drop the entries and fill on a change to another kind", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.BooleanT)
			Expect(normalize(ctx, readField("float", ch, map[string]any{
				"fill":        1,
				"enum_values": []any{map[string]any{"label": "ON", "value": 1}},
			}))).To(SatisfyAll(
				HaveKeyWithValue("type", "boolean"),
				Not(HaveKey("fill")),
				HaveKeyWithValue("enum_values", BeEmpty()),
			))
		})

		It("Should give a new timestamp field the ISO 8601 format", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.TimestampT)
			Expect(normalize(ctx, readField("float", ch, nil))).
				To(HaveKeyWithValue("time_format", "iso8601"))
		})

		It("Should keep the first entry of a repeated label", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.StringT)
			Expect(normalize(ctx, readField("string", ch, map[string]any{
				"enum_values": []any{
					map[string]any{"label": "A", "value": "x"},
					map[string]any{"label": "A", "value": "y"},
				},
			}))).To(HaveKeyWithValue("enum_values", HaveExactElements(
				map[string]any{"label": "A", "value": "x"},
			)))
		})

		It("Should import a v1 field as the kind its channel takes", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Int32T)
			data := MustSucceed(svc.Read.Normalize(ctx, db, 1, readConfig(
				map[string]any{
					"key":       "f1",
					"channel":   ch.Key(),
					"pointer":   "/v",
					"data_type": "float64",
					"enum_values": []any{
						map[string]any{"label": "ON", "value": 1},
						map[string]any{"label": "HALF", "value": 0.5},
					},
				},
			)))
			Expect(fieldsOf(data)[0]).To(SatisfyAll(
				HaveKeyWithValue("type", "integer"),
				Not(HaveKey("data_type")),
				HaveKeyWithValue("enum_values", HaveExactElements(
					map[string]any{"label": "ON", "value": "1"},
				)),
			))
		})

		It("Should keep the entries of an imported v1 field with a time format", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Uint8T)
			data := MustSucceed(svc.Read.Normalize(ctx, db, 1, readConfig(
				map[string]any{
					"key":         "f1",
					"channel":     ch.Key(),
					"pointer":     "/v",
					"time_format": "unix_sec",
					"enum_values": []any{map[string]any{"label": "ON", "value": 1}},
				},
			)))
			Expect(fieldsOf(data)[0]).To(SatisfyAll(
				HaveKeyWithValue("type", "unsigned"),
				HaveKeyWithValue("enum_values", HaveExactElements(
					map[string]any{"label": "ON", "value": "1"},
				)),
			))
		})

		It("Should import a v1 timestamp field with its time format", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.TimestampT)
			data := MustSucceed(svc.Read.Normalize(ctx, db, 1, readConfig(
				map[string]any{
					"key":         "f1",
					"channel":     ch.Key(),
					"pointer":     "/ts",
					"time_format": "unix_ms",
				},
			)))
			Expect(fieldsOf(data)[0]).To(SatisfyAll(
				HaveKeyWithValue("type", "timestamp"),
				HaveKeyWithValue("time_format", "unix_ms"),
			))
		})

		It(
			"Should import a v1 field on a boolean channel by the rules of Reconcile",
			func(
				ctx SpecContext,
			) {
				ch := createChannel(ctx, telem.BooleanT)
				data := MustSucceed(svc.Read.Normalize(ctx, db, 1, readConfig(
					map[string]any{
						"key":         "f1",
						"channel":     ch.Key(),
						"pointer":     "/v",
						"enum_values": []any{map[string]any{"label": "ON", "value": 1}},
					},
				)))
				Expect(fieldsOf(data)[0]).To(SatisfyAll(
					HaveKeyWithValue("type", "boolean"),
					HaveKeyWithValue("enum_values", BeEmpty()),
				))
			},
		)

		It("Should leave a field with no channel unchanged", func(ctx SpecContext) {
			Expect(normalize(ctx, readField("float", channel.Channel{}, map[string]any{
				"data_type": "float32",
			}))).To(SatisfyAll(
				HaveKeyWithValue("type", "float"),
				HaveKeyWithValue("data_type", "float32"),
			))
		})
	})
})
