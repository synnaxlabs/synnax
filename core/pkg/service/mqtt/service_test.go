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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		db    *gorp.DB
		svc   *mqtt.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's Core holds a mqtt service and a task service of its own.
		node := testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		svc = MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
			DB:      db,
			Channel: channels,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statuses,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the channel service", func(ctx SpecContext) {
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("channel: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per MQTT task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				mqtt.ReadTaskType,
				mqtt.WriteTaskType,
				mqtt.ScanTaskType,
				mqtt.EdgeTaskType,
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import a config at version 1 and reject a later version",
			func(ctx SpecContext, store func(*mqtt.Service) config.Store) {
				s := store(svc)
				Expect(s.Version()).To(BeEquivalentTo(1))
				MustSucceed(s.Normalize(ctx, db, 1, msgpack.EncodedJSON{}))
				newer := " version 2 is newer than this Core supports (latest: 1)"
				Expect(s.Normalize(ctx, db, 2, msgpack.EncodedJSON{})).Error().
					To(MatchError(ContainSubstring(s.Type() + newer)))
			},
			Entry("read", func(s *mqtt.Service) config.Store { return s.Read }),
			Entry("write", func(s *mqtt.Service) config.Store { return s.Write }),
			Entry("scan", func(s *mqtt.Service) config.Store { return s.Scan }),
			Entry("edge", func(s *mqtt.Service) config.Store { return s.Edge }),
		)
	})

	Describe("Write", func() {
		It("Should store a plain read entry with its schema defaults", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "broker-1")).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "broker-1",
				"entries": []any{map[string]any{
					"key":   "entry-1",
					"type":  "plain",
					"topic": "plant/line1/temperature",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("broker-1"))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("topic", "plant/line1/temperature"),
				HaveKeyWithValue("qos", "at_most_once"),
				HaveKeyWithValue("retained_ignored", false),
			)))
		})

		It("Should store a Sparkplug read entry", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"entries": []any{map[string]any{
					"key":       "entry-1",
					"type":      "sparkplug",
					"group":     "plant",
					"edge_node": "line1",
					"tag":       "temperature",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("edge_node", "line1"),
				HaveKeyWithValue("data_type", "float64"),
			)))
		})

		It("Should default a plain write target to QoS 1 and not retained", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"targets": []any{map[string]any{"key": "target-1", "type": "plain"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["targets"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("qos", "at_least_once"),
				HaveKeyWithValue("retained", false),
			)))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should default an edge node tag to the double type", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Edge, msgpack.EncodedJSON{
				"group":     "plant",
				"edge_node": "synnax",
				"tags":      []any{map[string]any{"key": "tag-1", "name": "pressure"}},
			}))
			data := MustSucceed(svc.Edge.Read(ctx, db, key))
			Expect(data["tags"]).To(HaveExactElements(
				HaveKeyWithValue("tag_type", "double"),
			))
		})

		It("Should return the validation error for an invalid QoS", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "plain", "qos": "twice"}},
			})).Error().
				To(MatchError(ContainSubstring("twice")))
		})

		It("Should return the validation error for an unknown entry type", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "amqp"}},
			})).Error().
				To(MatchError(ContainSubstring("amqp")))
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
})
