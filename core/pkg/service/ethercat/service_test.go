// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ethercat_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// openDevices opens a device service on db and creates three slaves on its embedded
// rack: "slave" with a unique input and output name, "el1008" with eight inputs named
// "Input", and "malformed" with PDOs that do not decode. It also returns the config
// that opens Tasks on the services it opens.
func openDevices(
	ctx SpecContext,
	db *gorp.DB,
) (*device.Service, tasktestutil.TasksConfig) {
	GinkgoHelper()
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         db,
		HostKey:    1,
		Status:     statusSvc,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
	}))
	devices := MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:     db,
		Status: statusSvc,
		Rack:   rackSvc,
	}))
	entry := func(name string, index int) map[string]any {
		return map[string]any{
			"name":       name,
			"index":      index,
			"sub_index":  1,
			"bit_length": 16,
			"data_type":  "int16",
		}
	}
	inputs := make([]any, 8)
	for i := range inputs {
		inputs[i] = entry("Input", 0x6000+i*0x10)
	}
	slaves := []device.Device{
		{
			Key:      "slave",
			Rack:     rackSvc.EmbeddedKey,
			Location: "eth0.1",
			Name:     "EL3102",
			Make:     "ethercat",
			Model:    "EL3102",
			Properties: msgpack.EncodedJSON{"pdos": map[string]any{
				"inputs":  []any{entry("Value", 0x6000)},
				"outputs": []any{entry("Output", 0x7000)},
			}},
		},
		{
			Key:        "el1008",
			Rack:       rackSvc.EmbeddedKey,
			Location:   "eth0.2",
			Name:       "EL1008",
			Make:       "ethercat",
			Model:      "EL1008",
			Properties: msgpack.EncodedJSON{"pdos": map[string]any{"inputs": inputs}},
		},
		{
			Key:        "malformed",
			Rack:       rackSvc.EmbeddedKey,
			Location:   "eth0.3",
			Name:       "EL2008",
			Make:       "ethercat",
			Model:      "EL2008",
			Properties: msgpack.EncodedJSON{"pdos": "garbage"},
		},
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return devices.NewWriter(tx).CreateMany(ctx, &slaves)
	})).To(Succeed())
	return devices, tasktestutil.TasksConfig{
		DB:     db,
		Status: statusSvc,
		Rack:   rackSvc,
		Device: devices,
	}
}

var _ = Describe("Service", func() {
	var (
		svc        *ethercat.Service
		devices    *device.Service
		channelSvc *channel.Service
		tasks      *tasktestutil.Tasks
		db         *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		var tasksCfg tasktestutil.TasksConfig
		devices, tasksCfg = openDevices(ctx, node.DB)
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  tasksCfg.Status,
		}))
		svc = MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
			DB:     node.DB,
			Device: devices,
		}))
		tasksCfg.Stores = svc.Stores()
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasksCfg))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(ethercat.OpenService(ctx, ethercat.ServiceConfig{
				Device: devices,
			})).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the device service", func(
			ctx SpecContext,
		) {
			Expect(ethercat.OpenService(ctx, ethercat.ServiceConfig{
				DB: db,
			})).Error().To(MatchError(ContainSubstring("device: must be non-nil")))
		})

		It("Should address the channels of stored version 1 configs", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			devices, _ := openDevices(ctx, db)
			var read v1.ReadConfig
			read.Key = uuid.New()
			read.Channels = []v1.ReadChannel{
				{Variant: v1.AutomaticReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "a", Device: "slave"},
					Pdo:             "Value",
				}},
				{Variant: v1.AutomaticReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "b", Device: "el1008"},
					Pdo:             "Input",
				}},
			}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, read)).
				To(Succeed())
			var write v1.WriteConfig
			write.Key = uuid.New()
			write.Channels = []v1.WriteChannel{{Variant: v1.AutomaticWriteChannel{
				BaseWriteChannel: v1.BaseWriteChannel{Key: "c", Device: "slave"},
				Pdo:              "Output",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, write)).
				To(Succeed())
			svc := MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
				DB:     db,
				Device: devices,
			}))
			readData := MustSucceed(svc.Read.Read(ctx, db, read.Key))
			Expect(readData["channels"]).To(Equal([]any{
				map[string]any{
					"type":       "entry",
					"key":        "a",
					"name":       "",
					"disabled":   false,
					"channel":    float64(0),
					"device":     "slave",
					"index":      float64(0x6000),
					"sub_index":  float64(1),
					"data_type":  "int16",
					"calibrated": false,
					"scale":      map[string]any{"type": "none"},
				},
				map[string]any{
					"type":     "unresolved",
					"key":      "b",
					"name":     "",
					"disabled": false,
					"channel":  float64(0),
					"device":   "el1008",
					"pdo":      "Input",
				},
			}))
			writeData := MustSucceed(svc.Write.Read(ctx, db, write.Key))
			channels := writeData["channels"].([]any)
			Expect(channels[0]).To(HaveKeyWithValue("type", "entry"))
			Expect(channels[0]).To(HaveKeyWithValue("index", float64(0x7000)))
			Expect(channels[0]).To(HaveKeyWithValue("stop_value", float64(0)))
		})

		It("Should address the channels on a slave a released Python client made", func(
			ctx SpecContext,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			var applied []gorp.Migration
			for _, key := range []string{
				"normalize_keys",
				"v0.status_backfill",
				"msgpack_to_orc",
				"v54_drop_status_parent",
				"v59_http_health_check_lists",
			} {
				applied = append(applied, migrate.New(
					key,
					func(context.Context, gorp.Tx, alamos.Instrumentation) error {
						return nil
					},
				))
			}
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: applied,
			})).To(Succeed())
			// The device a 0.59.2 Python ethercat.Device stored.
			slave := device.Device{
				Key:      "python",
				Location: "eth0:4",
				Name:     "EL3104",
				Make:     "EtherCAT",
				Model:    "Slave",
				Properties: msgpack.EncodedJSON{
					"network":  "eth0",
					"position": 4,
					"name":     "EL3104",
					"enabled":  true,
					"pdos": map[string]any{
						"inputs": []any{map[string]any{
							"name":       "Value",
							"pdo_index":  0x1A00,
							"index":      0x6000,
							"sub_index":  1,
							"bit_length": 16,
							"data_type":  "int16",
						}},
						"outputs": []any{},
					},
				},
			}
			Expect(gorp.NewCreate[device.Key, device.Device]().
				Entry(&slave).Exec(ctx, db)).To(Succeed())
			devices, _ := openDevices(ctx, db)
			var read v1.ReadConfig
			read.Key = uuid.New()
			read.Channels = []v1.ReadChannel{{Variant: v1.AutomaticReadChannel{
				BaseReadChannel: v1.BaseReadChannel{Key: "a", Device: "python"},
				Pdo:             "Value",
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, read)).
				To(Succeed())
			svc := MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
				DB:     db,
				Device: devices,
			}))
			readData := MustSucceed(svc.Read.Read(ctx, db, read.Key))
			channels := readData["channels"].([]any)
			Expect(channels[0]).To(HaveKeyWithValue("type", "entry"))
			Expect(channels[0]).To(HaveKeyWithValue("index", float64(0x6000)))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per EtherCAT task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"ethercat_read",
				"ethercat_write",
				"ethercat_scan",
			))
		})
	})

	Describe("Write", func() {
		readChannels := func(ctx SpecContext, channels ...any) []any {
			GinkgoHelper()
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"channels": channels,
			}))
			return MustSucceed(svc.Read.Read(ctx, db, key))["channels"].([]any)
		}

		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":      "entry",
					"key":       "chan-1",
					"device":    "slave",
					"channel":   ch,
					"index":     0x6000,
					"sub_index": 1,
					"data_type": "int16",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should address an unresolved read channel by its slave's input", func(
			ctx SpecContext,
		) {
			channels := readChannels(ctx, map[string]any{
				"type":   "unresolved",
				"key":    "a",
				"device": "slave",
				"pdo":    "Value",
			})
			Expect(channels[0]).To(HaveKeyWithValue("type", "entry"))
			Expect(channels[0]).To(HaveKeyWithValue("index", float64(0x6000)))
			Expect(channels[0]).To(HaveKeyWithValue("data_type", "int16"))
		})

		It("Should leave a read channel unresolved on a name the slave repeats", func(
			ctx SpecContext,
		) {
			channels := readChannels(ctx, map[string]any{
				"type":   "unresolved",
				"key":    "a",
				"device": "el1008",
				"pdo":    "Input",
			})
			Expect(channels[0]).To(HaveKeyWithValue("type", "unresolved"))
			Expect(channels[0]).To(HaveKeyWithValue("pdo", "Input"))
		})

		It("Should refuse a read channel on a missing device", func(ctx SpecContext) {
			Expect(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":   "unresolved",
					"key":    "a",
					"device": "gone",
					"pdo":    "Value",
				}},
			})).Error().To(MatchError(
				ContainSubstring("channels.device: device gone does not exist"),
			))
		})

		It("Should reject an unresolved channel on a slave with malformed PDOs", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":   "unresolved",
					"key":    "a",
					"device": "malformed",
					"pdo":    "Value",
				}},
			})).Error().To(SatisfyAll(
				MatchError(ContainSubstring("device malformed properties: json: ")),
				MatchError(ContainSubstring(`within "/pdos"`)),
			))
		})

		It("Should address an unresolved write channel by its slave's output", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":   "unresolved",
					"key":    "c",
					"device": "slave",
					"pdo":    "Output",
				}},
			}))
			channels := MustSucceed(svc.Write.Read(ctx, db, key))["channels"].([]any)
			Expect(channels[0]).To(HaveKeyWithValue("type", "entry"))
			Expect(channels[0]).To(HaveKeyWithValue("index", float64(0x7000)))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply write config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 25))
			Expect(data["execution_rate"]).To(BeNumerically("==", 1000))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})
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

	DescribeTable("Should reject a write config whose state rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"state_rate": rate,
			})).Error().To(MatchError(ContainSubstring("state_rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)

	DescribeTable("Should reject a write config whose execution rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"execution_rate": rate,
			})).Error().To(MatchError(ContainSubstring("execution_rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
