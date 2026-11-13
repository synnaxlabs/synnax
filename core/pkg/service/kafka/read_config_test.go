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
	"context"
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ReadConfig", func() {
	var (
		db       *gorp.DB
		svc      *kafka.Service
		channels *channel.Service
		index    channel.Channel
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
		index = channel.Channel{
			Name:     "index_" + uuid.New().String()[:8],
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &index)
		})).To(Succeed())
	})

	createChannel := func(ctx context.Context, dt telem.DataType) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:       "data_" + uuid.New().String()[:8],
			DataType:   dt,
			LocalIndex: index.LocalKey,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	configOf := func(fields ...map[string]any) msgpack.EncodedJSON {
		f := make([]any, len(fields))
		for i, fld := range fields {
			f[i] = fld
		}
		return msgpack.EncodedJSON{
			"records": []any{map[string]any{"key": "r", "fields": f}},
		}
	}

	fieldsOf := func(data msgpack.EncodedJSON) []any {
		return data["records"].([]any)[0].(map[string]any)["fields"].([]any)
	}

	normalize := func(
		ctx context.Context,
		data msgpack.EncodedJSON,
	) map[string]any {
		GinkgoHelper()
		var out msgpack.EncodedJSON
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			out, err = svc.Read.Normalize(ctx, tx, svc.Read.Version(), data)
			return err
		})).To(Succeed())
		return fieldsOf(out)[0].(map[string]any)
	}

	normalizeAt := func(
		ctx context.Context,
		version imex.Version,
		data msgpack.EncodedJSON,
	) map[string]any {
		GinkgoHelper()
		var out msgpack.EncodedJSON
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			out, err = svc.Read.Normalize(ctx, tx, version, data)
			return err
		})).To(Succeed())
		return fieldsOf(out)[0].(map[string]any)
	}

	prepare := func(ctx context.Context, data msgpack.EncodedJSON) error {
		_, err := svc.Read.Prepare(ctx, db, uuid.New(), "", data)
		return err
	}

	Describe("Validate", func() {
		It("Should reject a label that repeats", func(ctx SpecContext) {
			Expect(prepare(ctx, configOf(map[string]any{
				"key":  "f",
				"type": "unsigned",
				"enum_values": []any{
					map[string]any{"label": "ON", "value": "1"},
					map[string]any{"label": "ON", "value": "2"},
				},
			}))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`records.0.fields.0.enum_values: label "ON" repeats`,
				)),
			))
		})

		It("Should accept an empty label on an integer field", func(ctx SpecContext) {
			Expect(prepare(ctx, configOf(map[string]any{
				"key":         "f",
				"type":        "integer",
				"enum_values": []any{map[string]any{"label": "", "value": "0"}},
			}))).To(Succeed())
		})

		It("Should accept a value that repeats under two labels", func(
			ctx SpecContext,
		) {
			Expect(prepare(ctx, configOf(map[string]any{
				"key":  "f",
				"type": "unsigned",
				"enum_values": []any{
					map[string]any{"label": "ON", "value": "1"},
					map[string]any{"label": "HIGH", "value": "1"},
				},
			}))).To(Succeed())
		})

		It("Should reject the data type of another kind", func(ctx SpecContext) {
			Expect(prepare(ctx, configOf(map[string]any{
				"key":       "f",
				"type":      "integer",
				"data_type": "float32",
			}))).To(MatchError(ContainSubstring(
				"records.0.fields.0.data_type: float32 is not a data type of kind integer",
			)))
		})

		It("Should reject an entry that the data type of a new channel cannot hold",
			func(ctx SpecContext) {
				Expect(prepare(ctx, configOf(map[string]any{
					"key":       "f",
					"type":      "integer",
					"data_type": "int8",
					"enum_values": []any{
						map[string]any{"label": "BIG", "value": "300"},
					},
				}))).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("records.0.fields.0.enum_values.0.value"),
					),
				))
			},
		)

		It("Should reject a fill that the data type of a new channel cannot hold",
			func(ctx SpecContext) {
				Expect(prepare(ctx, configOf(map[string]any{
					"key":       "f",
					"type":      "unsigned",
					"data_type": "uint8",
					"fill":      "300",
				}))).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("records.0.fields.0.fill")),
				))
			},
		)

		It("Should reject a null field", func(ctx SpecContext) {
			Expect(prepare(ctx, msgpack.EncodedJSON{
				"records": []any{map[string]any{"key": "r", "fields": []any{nil}}},
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("records.0.fields.0")),
			))
		})

		It("Should save a draft bytes field with no pointer", func(ctx SpecContext) {
			Expect(prepare(ctx, configOf(
				map[string]any{"key": "raw", "type": "bytes"},
				map[string]any{"key": "f", "type": "float", "pointer": "/v"},
			))).To(Succeed())
		})
	})

	Describe("Resolve", func() {
		It("Should reject a field whose kind does not match its channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Float32T)
			Expect(prepare(ctx, configOf(map[string]any{
				"key":     "f",
				"type":    "integer",
				"channel": ch.Key(),
			}))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"field of kind integer does not match channel",
				)),
			))
		})

		It("Should reject an entry that does not fit the channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Int8T)
			Expect(prepare(ctx, configOf(map[string]any{
				"key":         "f",
				"type":        "integer",
				"channel":     ch.Key(),
				"enum_values": []any{map[string]any{"label": "BIG", "value": "200"}},
			}))).To(MatchError(ContainSubstring("200 is out of range for int8")))
		})

		It("Should reject a fill that does not fit the channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Int32T)
			Expect(prepare(ctx, configOf(map[string]any{
				"key":     "f",
				"type":    "integer",
				"channel": ch.Key(),
				"fill":    "-3000000000",
			}))).To(MatchError(ContainSubstring("fill: -3000000000 is out of range")))
		})

		It("Should reject a fill on an index channel", func(ctx SpecContext) {
			Expect(prepare(ctx, configOf(map[string]any{
				"key":         "f",
				"type":        "timestamp",
				"time_format": "unix_ns",
				"channel":     index.Key(),
				"fill":        "0",
			}))).To(MatchError(ContainSubstring("takes no fill")))
		})

		It("Should accept a mismatched field that is disabled", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Float32T)
			Expect(prepare(ctx, configOf(map[string]any{
				"key":      "f",
				"type":     "string",
				"channel":  ch.Key(),
				"disabled": true,
			}))).To(Succeed())
		})

		It("Should clear the data type of a field with a channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Float32T)
			rec := MustSucceed(svc.Read.Prepare(ctx, db, uuid.New(), "", configOf(
				map[string]any{
					"key":       "f",
					"type":      "float",
					"channel":   ch.Key(),
					"data_type": "float64",
				},
				map[string]any{"key": "g", "type": "float", "data_type": "float32"},
			)))
			fields := fieldsOf(MustSucceed(rec.Data()))
			Expect(fields[0]).NotTo(HaveKey("data_type"))
			Expect(fields[1]).To(HaveKeyWithValue("data_type", "float32"))
		})
	})

	Describe("Reconcile", func() {
		numeric := map[string]any{
			"key":     "f",
			"type":    "integer",
			"pointer": "/v",
			"enum_values": []any{
				map[string]any{"label": "OFF", "value": "0"},
				map[string]any{"label": "ON", "value": "1"},
			},
			"fill": "0",
		}

		DescribeTable("Should change the field to the kind of its channel",
			func(
				ctx SpecContext,
				dt telem.DataType,
				expected map[string]any,
			) {
				ch := createChannel(ctx, dt)
				fld := map[string]any{"channel": ch.Key()}
				maps.Copy(fld, numeric)
				out := normalize(ctx, configOf(fld))
				Expect(out).To(HaveKeyWithValue("pointer", "/v"))
				Expect(
					out,
				).To(HaveKeyWithValue("channel", BeNumerically("==", ch.Key())))
				for k, v := range expected {
					Expect(out).To(HaveKeyWithValue(k, v))
				}
				if _, ok := expected["fill"]; !ok {
					Expect(out).NotTo(HaveKey("fill"))
				}
			},
			Entry("float", telem.Float32T, map[string]any{
				"type": "float",
				"enum_values": []any{
					map[string]any{"label": "OFF", "value": 0.0},
					map[string]any{"label": "ON", "value": 1.0},
				},
				"fill": 0.0,
			}),
			Entry("integer", telem.Int16T, map[string]any{
				"type": "integer",
				"enum_values": []any{
					map[string]any{"label": "OFF", "value": "0"},
					map[string]any{"label": "ON", "value": "1"},
				},
				"fill": "0",
			}),
			Entry("unsigned", telem.Uint8T, map[string]any{
				"type": "unsigned",
				"enum_values": []any{
					map[string]any{"label": "OFF", "value": "0"},
					map[string]any{"label": "ON", "value": "1"},
				},
				"fill": "0",
			}),
			Entry("string", telem.StringT, map[string]any{
				"type":        "string",
				"enum_values": BeEmpty(),
			}),
			Entry("boolean, as value != 0", telem.BooleanT, map[string]any{
				"type": "boolean",
				"enum_values": []any{
					map[string]any{"label": "OFF", "value": false},
					map[string]any{"label": "ON", "value": true},
				},
				"fill": false,
			}),
			Entry("timestamp", telem.TimestampT, map[string]any{
				"type":        "timestamp",
				"time_format": "iso8601",
			}),
			Entry("json", telem.JSONT, map[string]any{"type": "json"}),
			Entry("uuid", telem.UUIDT, map[string]any{"type": "uuid"}),
			Entry("bytes", telem.BytesT, map[string]any{"type": "bytes"}),
		)

		It("Should drop entries and a fill that the channel does not hold",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Uint8T)
				out := normalize(ctx, configOf(map[string]any{
					"key":     "f",
					"type":    "float",
					"channel": ch.Key(),
					"enum_values": []any{
						map[string]any{"label": "HALF", "value": 0.5},
						map[string]any{"label": "ON", "value": 1},
						map[string]any{"label": "BIG", "value": 300},
						map[string]any{"label": "LOW", "value": -1},
						map[string]any{"label": "NONE", "value": "NaN"},
					},
					"fill": "NaN",
				}))
				Expect(out).To(HaveKeyWithValue("type", "unsigned"))
				Expect(out["enum_values"]).To(HaveExactElements(
					map[string]any{"label": "ON", "value": "1"},
				))
				Expect(out).NotTo(HaveKey("fill"))
			},
		)

		It(
			"Should drop entries and a fill that a channel of the same kind does not hold",
			func(ctx SpecContext) {
				ch := createChannel(ctx, telem.Int8T)
				out := normalize(ctx, configOf(map[string]any{
					"key":     "f",
					"type":    "integer",
					"channel": ch.Key(),
					"enum_values": []any{
						map[string]any{"label": "ON", "value": "1"},
						map[string]any{"label": "BIG", "value": "200"},
					},
					"fill": "-200",
				}))
				Expect(out).To(HaveKeyWithValue("type", "integer"))
				Expect(out["enum_values"]).To(HaveExactElements(
					map[string]any{"label": "ON", "value": "1"},
				))
				Expect(out).NotTo(HaveKey("fill"))
			},
		)

		It(
			"Should drop the fill of a field on an index channel",
			func(ctx SpecContext) {
				out := normalize(ctx, configOf(map[string]any{
					"key":         "f",
					"type":        "timestamp",
					"channel":     index.Key(),
					"time_format": "unix_ns",
					"fill":        "0",
				}))
				Expect(out).To(HaveKeyWithValue("time_format", "unix_ns"))
				Expect(out).NotTo(HaveKey("fill"))
			},
		)

		DescribeTable("Should drop integers that a float channel rounds",
			func(ctx SpecContext, dt telem.DataType, kept, rounded string) {
				ch := createChannel(ctx, dt)
				out := normalize(ctx, configOf(map[string]any{
					"key":     "f",
					"type":    "integer",
					"channel": ch.Key(),
					"enum_values": []any{
						map[string]any{"label": "KEPT", "value": kept},
						map[string]any{"label": "ROUNDED", "value": rounded},
					},
					"fill": rounded,
				}))
				Expect(out).To(HaveKeyWithValue("type", "float"))
				Expect(out["enum_values"]).To(HaveExactElements(
					HaveKeyWithValue("label", "KEPT"),
				))
				Expect(out).NotTo(HaveKey("fill"))
			},
			Entry("float64", telem.Float64T, "9007199254740992", "9007199254740993"),
			Entry("float32", telem.Float32T, "16777216", "16777217"),
		)

		It("Should lift a v0 field and change it to the kind of its channel", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Uint8T)
			out := normalizeAt(ctx, v0.Version, configOf(map[string]any{
				"key":         "f",
				"pointer":     "/v",
				"channel":     ch.Key(),
				"data_type":   "int16",
				"enum_values": []any{map[string]any{"label": "ON", "value": 1}},
			}))
			Expect(out).To(HaveKeyWithValue("type", "unsigned"))
			Expect(out).NotTo(HaveKey("data_type"))
			Expect(out["enum_values"]).To(HaveExactElements(
				map[string]any{"label": "ON", "value": "1"},
			))
		})

		It("Should keep the first entry of a repeated label", func(ctx SpecContext) {
			out := normalize(ctx, configOf(map[string]any{
				"key":  "f",
				"type": "unsigned",
				"enum_values": []any{
					map[string]any{"label": "ON", "value": "1"},
					map[string]any{"label": "HIGH", "value": "1"},
					map[string]any{"label": "ON", "value": "2"},
				},
			}))
			Expect(out["enum_values"]).To(HaveExactElements(
				map[string]any{"label": "ON", "value": "1"},
				map[string]any{"label": "HIGH", "value": "1"},
			))
		})

		It("Should keep an empty label on an integer field", func(ctx SpecContext) {
			out := normalize(ctx, configOf(map[string]any{
				"key":         "f",
				"type":        "integer",
				"enum_values": []any{map[string]any{"label": "", "value": "0"}},
			}))
			Expect(out["enum_values"]).To(HaveExactElements(
				map[string]any{"label": "", "value": "0"},
			))
		})

		It("Should clear the data type and keep a field that matches", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx, telem.Float32T)
			out := normalize(ctx, configOf(map[string]any{
				"key":         "f",
				"type":        "float",
				"channel":     ch.Key(),
				"data_type":   "float64",
				"enum_values": []any{map[string]any{"label": "ON", "value": 1.5}},
				"fill":        "Infinity",
			}))
			Expect(out).NotTo(HaveKey("data_type"))
			Expect(out).To(HaveKeyWithValue("fill", "Infinity"))
			Expect(out["enum_values"]).To(HaveLen(1))
		})

		It("Should keep the data type of a field with no channel", func(
			ctx SpecContext,
		) {
			out := normalize(ctx, configOf(map[string]any{
				"key":       "f",
				"type":      "integer",
				"data_type": "int16",
			}))
			Expect(out).To(HaveKeyWithValue("data_type", "int16"))
		})
	})
})
