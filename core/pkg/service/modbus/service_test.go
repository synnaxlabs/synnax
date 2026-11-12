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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	v1 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/status"
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
		svc        *modbus.Service
		channelSvc *svcchannel.Service
		tasks      *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		channelSvc = MustOpen(svcchannel.OpenService(ctx, svcchannel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		svc = MustOpen(modbus.OpenService(ctx, modbus.ServiceConfig{
			DB: db,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
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
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device":      "dev-1",
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "chan-1",
					"address": 3,
					"channel": ch,
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
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
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "uint16"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
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
					_, err := store().Prepare(ctx, db, key, "", msgpack.EncodedJSON{
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

		DescribeTable("Should refuse a register value the Driver cannot read",
			func(
				ctx SpecContext,
				store func() config.Store,
				channelType string,
				dataType string,
				msg string,
			) {
				Expect(store().Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type":          channelType,
						"key":           "chan-1",
						"data_type":     dataType,
						"string_length": 4,
					}},
				})).Error().To(MatchError(
					HaveSuffix("channels.0.data_type: " + msg),
				))
			},
			Entry(
				"0.60 holding register string read",
				func() config.Store { return svc.Read },
				"holding_register",
				"string",
				"must be a numeric data type, not string; read a string with a "+
					"register string channel",
			),
			Entry(
				"0.60 input register string read",
				func() config.Store { return svc.Read },
				"input_register",
				"string",
				"must be a numeric data type, not string; read a string with a "+
					"register string channel",
			),
			Entry(
				"holding register string write",
				func() config.Store { return svc.Write },
				"holding_register",
				"string",
				"must be a numeric data type, not string",
			),
			Entry(
				"timestamp read",
				func() config.Store { return svc.Read },
				"input_register",
				"timestamp",
				"must be a numeric data type, not timestamp",
			),
		)

		DescribeTable("Should accept every numeric register value type",
			func(ctx SpecContext, store func() config.Store, dataType string) {
				MustSucceed(
					store().Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
						"channels": []any{
							register(map[string]any{"data_type": dataType}),
						},
					}),
				)
			},
			Entry("int8 read", func() config.Store { return svc.Read }, "int8"),
			Entry("uint8 read", func() config.Store { return svc.Read }, "uint8"),
			Entry("int16 read", func() config.Store { return svc.Read }, "int16"),
			Entry("uint16 read", func() config.Store { return svc.Read }, "uint16"),
			Entry("int32 read", func() config.Store { return svc.Read }, "int32"),
			Entry("uint32 read", func() config.Store { return svc.Read }, "uint32"),
			Entry("int64 read", func() config.Store { return svc.Read }, "int64"),
			Entry("uint64 read", func() config.Store { return svc.Read }, "uint64"),
			Entry("float32 read", func() config.Store { return svc.Read }, "float32"),
			Entry("float64 read", func() config.Store { return svc.Read }, "float64"),
			Entry("int8 write", func() config.Store { return svc.Write }, "int8"),
			Entry("uint8 write", func() config.Store { return svc.Write }, "uint8"),
			Entry("int16 write", func() config.Store { return svc.Write }, "int16"),
			Entry("uint16 write", func() config.Store { return svc.Write }, "uint16"),
			Entry("int32 write", func() config.Store { return svc.Write }, "int32"),
			Entry("uint32 write", func() config.Store { return svc.Write }, "uint32"),
			Entry("int64 write", func() config.Store { return svc.Write }, "int64"),
			Entry("uint64 write", func() config.Store { return svc.Write }, "uint64"),
			Entry("float32 write", func() config.Store { return svc.Write }, "float32"),
			Entry("float64 write", func() config.Store { return svc.Write }, "float64"),
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
			ch := channel(MustSucceed(svc.Read.Read(ctx, db, key)))
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
			ch := channel(MustSucceed(svc.Write.Read(ctx, db, key)))
			Expect(ch).To(HaveKeyWithValue("bytes_swapped", false))
			Expect(ch).To(HaveKeyWithValue("words_swapped", true))
		})

		DescribeTable(
			"Should make a v1 false follow the device and keep a v1 true",
			func(ctx SpecContext, store func() config.Store) {
				ch := channel(MustSucceed(store().Normalize(ctx, db,
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
				MustSucceed(svc.Read.Normalize(ctx, db, 0, msgpack.EncodedJSON{
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
			chs := MustSucceed(migrated.Read.Read(ctx, stored, key))["channels"].([]any)
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
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
			out := MustSucceed(svc.Read.Normalize(ctx, db, svc.Read.Version(),
				msgpack.EncodedJSON{
					"device": "missing",
					"channels": []any{
						map[string]any{"type": "coil", "key": "a", "channel": ch},
						map[string]any{"type": "coil", "key": "b", "channel": 4244},
					},
				},
			))
			Expect(out).To(HaveKeyWithValue("device", ""))
			Expect(out["channels"]).To(HaveExactElements(
				SatisfyAll(
					HaveKeyWithValue("key", "a"),
					HaveKeyWithValue("channel", BeEquivalentTo(ch)),
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
				stored := testutil.NewNode(ctx)
				statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
					DB: stored.DB,
				}))
				channelSvc := MustOpen(svcchannel.OpenService(
					ctx,
					svcchannel.ServiceConfig{
						Channel: stored.Channel,
						DB:      stored.DB,
						HostKey: stored.Identity.HostKey,
						Status:  statusSvc,
					},
				))
				targets := MustOpen(
					tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
						DB:     stored.DB,
						Status: statusSvc,
					}),
				)
				ch := MustSucceed(
					channeltestutil.CreateVirtual(ctx, stored.DB, channelSvc, 1),
				)[0]
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
								Channel: ch,
							},
						}},
					},
				}
				read.SetKey(key)
				write := modbus.WriteConfig{Device: "kept"}
				write.SetKey(key)
				Expect(gorp.NewCreate[uuid.UUID, modbus.ReadConfig]().
					Entry(&read).Exec(ctx, stored.DB)).To(Succeed())
				Expect(gorp.NewCreate[uuid.UUID, modbus.WriteConfig]().
					Entry(&write).Exec(ctx, stored.DB)).To(Succeed())
				opened := MustOpen(
					modbus.OpenService(ctx, modbus.ServiceConfig{DB: stored.DB}),
				)
				data := MustSucceed(opened.Read.Read(ctx, stored.DB, key))
				Expect(data).To(HaveKeyWithValue("device", ""))
				Expect(data["channels"]).To(HaveExactElements(
					HaveKeyWithValue("channel", BeEquivalentTo(0)),
					HaveKeyWithValue("channel", BeEquivalentTo(ch)),
				))
				Expect(MustSucceed(opened.Write.Read(ctx, stored.DB, key))).
					To(HaveKeyWithValue("device", "kept"))
			},
		)
	})

	DescribeTable("Should reject a scan config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Scan.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)

	DescribeTable(
		"Should reject a read config with an invalid rate",
		func(ctx SpecContext, sampleRate, streamRate float64, msg string) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"sample_rate": sampleRate,
				"stream_rate": streamRate,
			})).Error().To(MatchError(ContainSubstring(msg)))
		},
		Entry("zero sample rate", 0.0, 0.0, "sample_rate: must be greater than 0"),
		Entry(
			"negative sample rate",
			-1.0,
			-2.0,
			"sample_rate: must be greater than 0",
		),
		Entry("zero stream rate", 10.0, 0.0, "stream_rate: must be greater than 0"),
		Entry(
			"negative stream rate",
			10.0,
			-1.0,
			"stream_rate: must be greater than 0",
		),
	)

	// The Console saves each field edit, so a draft can hold a stream rate above its
	// sample rate. The Driver refuses it on start.
	It("Should store a read config whose stream rate is above its sample rate",
		func(ctx SpecContext) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"sample_rate": 10.0,
				"stream_rate": 20.0,
			})).Error().ToNot(HaveOccurred())
		},
	)
})
