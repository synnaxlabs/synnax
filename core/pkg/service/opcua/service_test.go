// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opcua_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		svc        *opcua.Service
		channelSvc *channel.Service
		tasks      *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		svc = MustOpen(opcua.OpenService(ctx, opcua.ServiceConfig{
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
			Expect(opcua.OpenService(ctx, opcua.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Normalize", func() {
		DescribeTable("Should lift an unversioned read config to array sampling",
			func(ctx SpecContext, in msgpack.EncodedJSON, size int) {
				out := MustSucceed(svc.Read.Normalize(ctx, db, 0, in))
				Expect(out["sampling"]).To(And(
					HaveKeyWithValue("type", "array"),
					HaveKeyWithValue("size", BeNumerically("==", size)),
				))
			},
			Entry("a present array sampling",
				msgpack.EncodedJSON{
					"array_mode": false,
					"sampling":   map[string]any{"type": "array", "size": 24},
				},
				24,
			),
			Entry("a present array sampling below the minimum size",
				msgpack.EncodedJSON{
					"sampling": map[string]any{"type": "array", "size": 0},
				},
				1,
			),
			Entry("a present array sampling with a negative size",
				msgpack.EncodedJSON{
					"sampling": map[string]any{"type": "array", "size": -4},
				},
				1,
			),
			Entry("the flat array fields",
				msgpack.EncodedJSON{"arrayMode": true, "arraySize": 24},
				24,
			),
			Entry("the flat array fields below the minimum size",
				msgpack.EncodedJSON{"arrayMode": true, "arraySize": -4},
				1,
			),
		)

		It("Should reject an unversioned read config with an unknown sampling", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Normalize(ctx, db, 0, msgpack.EncodedJSON{
				"sampling": map[string]any{"type": "burst"},
			})).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per OPC UA task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"opc_read",
				"opc_write",
				"opc_scan",
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
					"key":     "chan-1",
					"node_id": "NS=2;I=8",
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
			Expect(data["sample_rate"]).To(BeNumerically("==", 50))
			Expect(data["stream_rate"]).To(BeNumerically("==", 25))
			Expect(data["sampling"]).To(Equal(map[string]any{"type": "unary"}))
		})

		It("Should apply write channel schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"key": "chan-1"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "float32"),
			))
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

	It("Should accept a zero stream rate under array sampling", func(ctx SpecContext) {
		Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
			"sample_rate": 10,
			"stream_rate": 0,
			"sampling":    map[string]any{"type": "array", "size": 1},
		})).Error().ToNot(HaveOccurred())
	})
})
