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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *opcua.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(opcua.OpenService(ctx, opcua.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(opcua.OpenService(ctx, opcua.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
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
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device":      "dev-1",
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"key":     "chan-1",
					"node_id": "NS=2;I=8",
					"channel": 42,
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 50))
			Expect(data["stream_rate"]).To(BeNumerically("==", 25))
			Expect(data["array_size"]).To(BeNumerically("==", 1))
		})

		It("Should apply write channel schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"key": "chan-1"}},
			})).To(Succeed())
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "float32"),
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
	})
})
