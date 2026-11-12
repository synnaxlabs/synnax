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
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel fields", func() {
	var (
		db       *gorp.DB
		svc      *mqtt.Service
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
		svc = MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
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

	field := func(kind string, ch channel.Key, entries ...map[string]any) map[string]any {
		f := map[string]any{
			"type":      kind,
			"channel":   ch,
			"data_type": "float64",
		}
		if len(entries) > 0 {
			list := make([]any, len(entries))
			for i, e := range entries {
				list[i] = e
			}
			f["enum_values"] = list
		}
		if kind == "timestamp" {
			f["time_format"] = "unix_ms"
		}
		return f
	}

	entry := func(label string, value any) map[string]any {
		return map[string]any{"label": label, "value": value}
	}

	config := func(fields ...map[string]any) msgpack.EncodedJSON {
		targets := make([]any, len(fields))
		for i, f := range fields {
			targets[i] = map[string]any{
				"key":     "t" + strconv.Itoa(i),
				"type":    "plain",
				"topic":   "plant/x",
				"channel": f,
			}
		}
		return msgpack.EncodedJSON{"targets": targets}
	}

	prepare := func(ctx SpecContext, data msgpack.EncodedJSON) (msgpack.EncodedJSON, error) {
		rec, err := svc.Write.Prepare(ctx, db, uuid.New(), data)
		if err != nil {
			return nil, err
		}
		return rec.Data()
	}

	firstField := func(cfg msgpack.EncodedJSON) map[string]any {
		GinkgoHelper()
		targets, ok := cfg["targets"].([]any)
		Expect(ok).To(BeTrue())
		Expect(targets).ToNot(BeEmpty())
		target, ok := targets[0].(map[string]any)
		Expect(ok).To(BeTrue())
		f, ok := target["channel"].(map[string]any)
		Expect(ok).To(BeTrue())
		return f
	}

	Describe("Prepare", func() {
		DescribeTable("Should accept each kind a channel's data type allows",
			func(ctx SpecContext, dt telem.DataType, kind string) {
				ch := create(ctx, dt)
				Expect(prepare(ctx, config(field(kind, ch)))).Error().
					ToNot(HaveOccurred())
			},
			Entry("float as float", telem.Float32T, "float"),
			Entry("float as string", telem.Float32T, "string"),
			Entry("float as boolean", telem.Float32T, "boolean"),
			Entry("boolean as boolean", telem.BooleanT, "boolean"),
			Entry("boolean as unsigned", telem.BooleanT, "unsigned"),
			Entry("boolean as string", telem.BooleanT, "string"),
			Entry("timestamp", telem.TimestampT, "timestamp"),
			Entry("json", telem.JSONT, "json"),
		)

		It("Should reject a kind a channel's data type does not allow",
			func(ctx SpecContext) {
				ch := create(ctx, telem.Int8T)
				Expect(
					prepare(ctx, config(field("unsigned", ch))),
				).Error().
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							"targets.0.channel: a int8 channel cannot be written as unsigned",
						)),
					))
			},
		)

		It(
			"Should reject an entry that does not fit the channel",
			func(ctx SpecContext) {
				ch := create(ctx, telem.Int8T)
				Expect(prepare(ctx, config(
					field("integer", ch, entry("BIG", "200")),
				))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						`enum label "BIG": 200 is out of range for int8`,
					)),
				))
			},
		)

		It(
			"Should reject a timestamp field with no time format",
			func(ctx SpecContext) {
				f := field("timestamp", create(ctx, telem.TimestampT))
				delete(f, "time_format")
				Expect(prepare(ctx, config(f))).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						`targets.0.channel: unknown time format ""`,
					)),
				))
			},
		)

		It(
			"Should narrow a float entry and clear the data type",
			func(ctx SpecContext) {
				ch := create(ctx, telem.Float32T)
				f := firstField(MustSucceed(prepare(ctx, config(
					field("float", ch, entry("TENTH", 0.1)),
				))))
				Expect(f["enum_values"]).To(HaveExactElements(HaveKeyWithValue(
					"value", BeNumerically("==", float64(float32(0.1))),
				)))
				Expect(f).ToNot(HaveKey("data_type"))
			},
		)

		It(
			"Should keep the data type of a field with no channel",
			func(ctx SpecContext) {
				f := firstField(MustSucceed(prepare(ctx, config(field("float", 0)))))
				Expect(f).To(HaveKeyWithValue("data_type", "float64"))
			},
		)

		It(
			"Should skip a disabled target and a missing channel",
			func(ctx SpecContext) {
				cfg := config(
					field("float", create(ctx, telem.StringT)),
					field("float", 999999),
				)
				cfg["targets"].([]any)[0].(map[string]any)["disabled"] = true
				Expect(prepare(ctx, cfg)).Error().ToNot(HaveOccurred())
			},
		)
	})

	Describe("Validate", func() {
		It("Should reject two entries with one stored value", func(ctx SpecContext) {
			Expect(prepare(ctx, config(
				field("float", 0, entry("A", "NaN"), entry("B", "NaN")),
			))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"targets.0.channel: enum_values[1]: value is already labeled",
				)),
			))
		})
	})

	Describe("Normalize", func() {
		It("Should set the kind of the channel's data type", func(ctx SpecContext) {
			cfg := config(field("float", create(ctx, telem.Int32T),
				entry("ONE", 1.0), entry("HALF", 0.5),
			))
			f := firstField(MustSucceed(
				svc.Write.Normalize(ctx, db, svc.Write.Version(), cfg),
			))
			Expect(f).To(HaveKeyWithValue("type", "integer"))
			Expect(f["enum_values"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("label", "ONE"),
				HaveKeyWithValue("value", "1"),
			)))
			Expect(f).ToNot(HaveKey("data_type"))
		})

		It("Should import a version 1 field as the kind of its channel", func(
			ctx SpecContext,
		) {
			old := map[string]any{
				"channel":   create(ctx, telem.Int8T),
				"json_type": "number",
				"data_type": "int8",
			}
			f := firstField(MustSucceed(svc.Write.Normalize(ctx, db, 1, config(old))))
			Expect(f).To(HaveKeyWithValue("type", "integer"))
			Expect(f).ToNot(HaveKey("json_type"))
		})

		It("Should keep the kind of a channel that does not exist", func(
			ctx SpecContext,
		) {
			f := firstField(MustSucceed(svc.Write.Normalize(
				ctx, db, svc.Write.Version(), config(field("float", 999999)),
			)))
			Expect(f).To(HaveKeyWithValue("type", "float"))
		})
	})
})

var _ = Describe("Stored write configs", func() {
	It(
		"Should lift v1 channel fields to the kinds their channels allow",
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
			plain := func(key string, f v1.ChannelField) v1.WriteTarget {
				return v1.WriteTarget{Variant: v1.PlainWriteTarget{
					BaseWriteTarget: v1.BaseWriteTarget{Key: key},
					Topic:           "plant/" + key,
					Channel:         f,
				}}
			}
			old := v1.WriteConfig{Key: uuid.New(), Targets: []v1.WriteTarget{
				plain("a", v1.ChannelField{
					Channel:  create(telem.Uint8T),
					JSONType: payload.JSONTypeNumber,
				}),
				plain("b", v1.ChannelField{
					Channel:    create(telem.Int16T),
					DataType:   telem.Int16T,
					JSONType:   payload.JSONTypeString,
					EnumValues: []payload.EnumEntry{{Label: "ON", Value: 1}},
				}),
				plain("c", v1.ChannelField{
					DataType: telem.Int32T,
					JSONType: payload.JSONTypeNumber,
				}),
			}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			svc := MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
				DB:      db,
				Channel: channels,
			}))
			var stored []v2.WriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v2.WriteConfig]().
				Entries(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored).To(HaveExactElements(HaveField("Key", old.Key)))
			data := MustSucceed(svc.Write.Read(ctx, db, old.Key))
			field := func(t any) any {
				return t.(map[string]any)["channel"]
			}
			targets := data["targets"].([]any)
			Expect(targets).To(HaveLen(3))
			Expect(field(targets[0])).To(SatisfyAll(
				HaveKeyWithValue("type", "unsigned"),
				Not(HaveKey("data_type")),
			))
			Expect(field(targets[1])).To(SatisfyAll(
				HaveKeyWithValue("type", "integer"),
				HaveKeyWithValue("enum_values", HaveExactElements(SatisfyAll(
					HaveKeyWithValue("label", "ON"),
					HaveKeyWithValue("value", "1"),
				))),
			))
			Expect(field(targets[2])).To(SatisfyAll(
				HaveKeyWithValue("type", "integer"),
				HaveKeyWithValue("data_type", "int32"),
			))
		},
	)
})
