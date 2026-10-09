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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *kafka.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{DB: db}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(kafka.OpenService(ctx, kafka.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per Kafka task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				kafka.ReadTaskType,
				kafka.WriteTaskType,
				kafka.ScanTaskType,
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device": "dev-1",
				"topic":  "telemetry",
				"fields": []any{map[string]any{
					"key":        "f-1",
					"pointer":    "/value",
					"record_key": "press_1",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["topic"]).To(Equal("telemetry"))
			Expect(data["fields"]).To(HaveExactElements(
				HaveKeyWithValue("record_key", "press_1"),
			))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"fields": []any{map[string]any{"key": "f-1"}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["start_offset"]).To(Equal("latest"))
			Expect(data["fields"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "float64"),
			))
		})

		It("Should apply write config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"key": "ch-1"}},
			})).To(Succeed())
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["record_key"]).To(Equal("channel_name"))
			Expect(data["record"]).To(And(
				HaveKeyWithValue("value_pointer", "/value"),
				HaveKeyWithValue("channel_pointer", "/channel"),
				HaveKeyWithValue("timestamp_pointer", "/timestamp"),
				HaveKeyWithValue("time_format", "unix_ns"),
			))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("json_type", "number"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Scan.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Scan.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should return the read validation error for an invalid start offset",
			func(ctx SpecContext) {
				Expect(svc.Read.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"start_offset": "bogus",
				})).To(MatchError(ContainSubstring("invalid start_offset: bogus")))
			},
		)

		It("Should return the write validation error for an invalid record key",
			func(ctx SpecContext) {
				Expect(svc.Write.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"record_key": "bogus",
				})).To(MatchError(ContainSubstring("invalid record_key: bogus")))
			},
		)
	})
})
