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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		otg *ontology.Ontology
		svc *ethercat.Service
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(ethercat.OpenService(ctx, ethercat.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	Describe("Stores", func() {
		It("Should expose one store per EtherCAT task type", func() {
			types := []ontology.ResourceType{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				ontology.ResourceTypeEthercatRead,
				ontology.ResourceTypeEthercatWrite,
				ontology.ResourceTypeEthercatScan,
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
	})
})
