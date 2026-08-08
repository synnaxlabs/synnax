// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/opc"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		otg *ontology.Ontology
		svc *opc.Service
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(opc.OpenService(ctx, opc.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	Describe("Stores", func() {
		It("Should expose one store per OPC UA task type", func() {
			types := []ontology.ResourceType{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				ontology.ResourceTypeOpcRead,
				ontology.ResourceTypeOpcWrite,
				ontology.ResourceTypeOpcScan,
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
	})
})
