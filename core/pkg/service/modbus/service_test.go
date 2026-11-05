// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package modbus_test

import (
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	v1 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// register returns a holding register read channel carrying the given extra fields.
func register(fields map[string]any) map[string]any {
	ch := map[string]any{"type": "holding_register", "key": "chan-1", "address": 2}
	maps.Copy(ch, fields)
	return ch
}

// channel returns the only channel of a normalized or stored config.
func channel(cfg msgpack.EncodedJSON) map[string]any {
	GinkgoHelper()
	chs := cfg["channels"].([]any)
	Expect(chs).To(HaveLen(1))
	return chs[0].(map[string]any)
}

var _ = Describe("Service", func() {
	var (
		svc   *modbus.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(modbus.OpenService(ctx, modbus.ServiceConfig{
			DB: db,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(modbus.OpenService(ctx, modbus.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per Modbus task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"modbus_read",
				"modbus_write",
				"modbus_scan",
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			Expect(tasks.Channels.Create(ctx, 42)).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device":      "dev-1",
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "chan-1",
					"address": 3,
					"channel": 42,
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply write channel schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "holding_register",
					"key":  "chan-1",
				}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "uint16"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		DescribeTable(
			"Should reject a register bit past 15",
			func(
				ctx SpecContext,
				store func() config.Store,
				channelType string,
			) {
				key := uuid.New()
				write := func(bit int) error {
					_, err := store().Prepare(ctx, nil, key, msgpack.EncodedJSON{
						"channels": []any{map[string]any{
							"type":    channelType,
							"key":     "chan-1",
							"address": 4,
							"bit":     bit,
						}},
					})
					return err
				}
				Expect(write(15)).To(Succeed())
				Expect(write(16)).To(MatchError(ContainSubstring(
					"channels.0.bit: must be less than or equal to 15",
				)))
			},
			Entry(
				"holding register bit read",
				func() config.Store { return svc.Read },
				"holding_register_bit",
			),
			Entry(
				"input register bit read",
				func() config.Store { return svc.Read },
				"input_register_bit",
			),
			Entry(
				"holding register bit write",
				func() config.Store { return svc.Write },
				"holding_register_bit",
			),
		)
	})

	Describe("Byte and word order", func() {
		It("Should export read and write configs above the v1 shape's version", func() {
			Expect(svc.Read.Version()).To(BeNumerically(">", v1.Version))
			Expect(svc.Write.Version()).To(BeNumerically(">", v1.Version))
		})

		It("Should store a channel with no override without swap fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"channels": []any{register(nil)},
			}))
			ch := channel(MustSucceed(svc.Read.Read(ctx, nil, key)))
			Expect(ch).ToNot(HaveKey("bytes_swapped"))
			Expect(ch).ToNot(HaveKey("words_swapped"))
		})

		It("Should keep an explicit override, false included", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"channels": []any{register(map[string]any{
					"bytes_swapped": false,
					"words_swapped": true,
				})},
			}))
			ch := channel(MustSucceed(svc.Write.Read(ctx, nil, key)))
			Expect(ch).To(HaveKeyWithValue("bytes_swapped", false))
			Expect(ch).To(HaveKeyWithValue("words_swapped", true))
		})

		DescribeTable(
			"Should make a v1 false follow the device and keep a v1 true",
			func(ctx SpecContext, store func() config.Store) {
				ch := channel(MustSucceed(store().Normalize(ctx, nil,
					v1.Version,
					msgpack.EncodedJSON{"channels": []any{register(map[string]any{
						"bytes_swapped": false,
						"words_swapped": true,
					})}},
				)))
				Expect(ch).ToNot(HaveKey("bytes_swapped"))
				Expect(ch).To(HaveKeyWithValue("words_swapped", true))
			},
			Entry("read", func() config.Store { return svc.Read }),
			Entry("write", func() config.Store { return svc.Write }),
		)

		It("Should make a released Console's swap_bytes false follow the device", func(
			ctx SpecContext,
		) {
			ch := channel(
				MustSucceed(svc.Read.Normalize(ctx, nil, 0, msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type":       "holding_register_input",
						"key":        "chan-1",
						"swap_bytes": false,
						"swap_words": true,
					}},
				})),
			)
			Expect(ch).To(HaveKeyWithValue("type", "holding_register"))
			Expect(ch).ToNot(HaveKey("bytes_swapped"))
			Expect(ch).To(HaveKeyWithValue("words_swapped", true))
		})

		It("Should migrate a stored v1 record when the service opens", func(
			ctx SpecContext,
		) {
			stored := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[uuid.UUID, v1.ReadConfig]{DB: stored},
			))
			key := uuid.New()
			old := v1.ReadConfig{Channels: []v1.ReadChannel{
				{Variant: v1.HoldingRegisterReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "a"},
					RegisterValue:   v1.RegisterValue{DataType: "float32"},
				}},
				{Variant: v1.InputRegisterReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "b"},
					RegisterValue: v1.RegisterValue{
						DataType:     "float32",
						BytesSwapped: true,
					},
				}},
			}}
			old.SetKey(key)
			Expect(gorp.NewCreate[uuid.UUID, v1.ReadConfig]().
				Entry(&old).Exec(ctx, stored)).To(Succeed())
			migrated := MustOpen(modbus.OpenService(
				ctx, modbus.ServiceConfig{DB: stored},
			))
			chs := MustSucceed(migrated.Read.Read(ctx, nil, key))["channels"].([]any)
			Expect(chs).To(HaveLen(2))
			Expect(chs[0]).ToNot(HaveKey("bytes_swapped"))
			Expect(chs[0]).ToNot(HaveKey("words_swapped"))
			Expect(chs[1]).To(HaveKeyWithValue("bytes_swapped", true))
			Expect(chs[1]).ToNot(HaveKey("words_swapped"))
		})
	})
	Describe("References", func() {
		It("Should refuse a config that names a channel that does not exist", func(
			ctx SpecContext,
		) {
			Expect(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "chan-1",
					"channel": 4242,
				}},
			})).Error().To(MatchError(
				ContainSubstring("channels.channel: channel 4242 does not exist"),
			))
		})

		It("Should refuse a config that names a device that does not exist", func(
			ctx SpecContext,
		) {
			Expect(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"device": "missing",
			})).Error().To(MatchError(
				ContainSubstring("device: device missing does not exist"),
			))
		})

		It("Should drop the channels and devices that do not exist on normalize", func(
			ctx SpecContext,
		) {
			Expect(tasks.Channels.Create(ctx, 4243)).To(Succeed())
			out := MustSucceed(svc.Read.Normalize(ctx, nil, svc.Read.Version(),
				msgpack.EncodedJSON{
					"device": "missing",
					"channels": []any{
						map[string]any{"type": "coil", "key": "a", "channel": 4243},
						map[string]any{"type": "coil", "key": "b", "channel": 4244},
					},
				},
			))
			Expect(out).To(HaveKeyWithValue("device", ""))
			Expect(out["channels"]).To(HaveExactElements(
				SatisfyAll(
					HaveKeyWithValue("key", "a"),
					HaveKeyWithValue("channel", BeEquivalentTo(4243)),
				),
				SatisfyAll(
					HaveKeyWithValue("key", "b"),
					HaveKeyWithValue("channel", BeEquivalentTo(0)),
				),
			))
		})

		It(
			"Should reset the stored channels and devices that do not exist on open",
			func(
				ctx SpecContext,
			) {
				stored := DeferClose(gorp.Wrap(memkv.New()))
				targets := MustOpen(
					tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
						DB: stored,
					}),
				)
				Expect(targets.Channels.Create(ctx, 10)).To(Succeed())
				Expect(targets.CreateDevice(ctx, "kept")).To(Succeed())
				key := uuid.New()
				read := modbus.ReadConfig{
					Device: "gone",
					Channels: []modbus.ReadChannel{
						{Variant: modbus.CoilReadChannel{
							BaseReadChannel: modbus.BaseReadChannel{
								Key:     "a",
								Channel: 9,
							},
						}},
						{Variant: modbus.CoilReadChannel{
							BaseReadChannel: modbus.BaseReadChannel{
								Key:     "b",
								Channel: 10,
							},
						}},
					},
				}
				read.SetKey(key)
				write := modbus.WriteConfig{Device: "kept"}
				write.SetKey(key)
				Expect(gorp.NewCreate[uuid.UUID, modbus.ReadConfig]().
					Entry(&read).Exec(ctx, stored)).To(Succeed())
				Expect(gorp.NewCreate[uuid.UUID, modbus.WriteConfig]().
					Entry(&write).Exec(ctx, stored)).To(Succeed())
				opened := MustOpen(
					modbus.OpenService(ctx, modbus.ServiceConfig{DB: stored}),
				)
				data := MustSucceed(opened.Read.Read(ctx, nil, key))
				Expect(data).To(HaveKeyWithValue("device", ""))
				Expect(data["channels"]).To(HaveExactElements(
					HaveKeyWithValue("channel", BeEquivalentTo(0)),
					HaveKeyWithValue("channel", BeEquivalentTo(10)),
				))
				Expect(MustSucceed(opened.Write.Read(ctx, nil, key))).
					To(HaveKeyWithValue("device", "kept"))
			},
		)
	})
})
