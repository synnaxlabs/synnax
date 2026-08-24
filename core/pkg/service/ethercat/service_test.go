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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *ethercat.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(ethercat.OpenService(ctx, ethercat.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
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
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":    "automatic",
					"key":     "chan-1",
					"device":  "dev-1",
					"channel": 42,
					"pdo":     "Inputs.Ch1",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply write config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 25))
			Expect(data["execution_rate"]).To(BeNumerically("==", 1000))
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
