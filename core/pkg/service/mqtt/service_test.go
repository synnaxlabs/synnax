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
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *mqtt.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{DB: db}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
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

	Describe("Write", func() {
		It("Should store a plain read entry with its schema defaults", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device": "broker-1",
				"entries": []any{map[string]any{
					"key":   "entry-1",
					"type":  "plain",
					"topic": "plant/line1/temperature",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("broker-1"))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("topic", "plant/line1/temperature"),
				HaveKeyWithValue("qos", "at_most_once"),
				HaveKeyWithValue("retained_ignored", false),
			)))
		})

		It("Should store a Sparkplug read entry", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"entries": []any{map[string]any{
					"key":       "entry-1",
					"type":      "sparkplug",
					"group":     "plant",
					"edge_node": "line1",
					"tag":       "temperature",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("edge_node", "line1"),
				HaveKeyWithValue("data_type", "float64"),
			)))
		})

		It("Should default a plain write target to QoS 1 and not retained", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"targets": []any{map[string]any{"key": "target-1", "type": "plain"}},
			})).To(Succeed())
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["targets"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("qos", "at_least_once"),
				HaveKeyWithValue("retained", false),
			)))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Scan.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Scan.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should default an edge node tag to the double type", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Edge.Write(ctx, nil, key, msgpack.EncodedJSON{
				"group":     "plant",
				"edge_node": "synnax",
				"tags":      []any{map[string]any{"key": "tag-1", "name": "pressure"}},
			})).To(Succeed())
			data := MustSucceed(svc.Edge.Read(ctx, nil, key))
			Expect(data["tags"]).To(HaveExactElements(
				HaveKeyWithValue("sparkplug_type", "double"),
			))
		})

		It("Should return the validation error for an invalid QoS", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "plain", "qos": "twice"}},
			})).To(MatchError(ContainSubstring("twice")))
		})

		It("Should return the validation error for an unknown entry type", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "amqp"}},
			})).To(MatchError(ContainSubstring("amqp")))
		})
	})
})
