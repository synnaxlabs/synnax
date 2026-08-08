// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		otg *ontology.Ontology
		svc *pd.Service
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	Describe("Stores", func() {
		It("Should expose the alert store", func() {
			types := []ontology.ResourceType{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(ontology.ResourceTypePagerdutyAlert))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded alert config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-1",
				"auto_start":  true,
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": "status-1",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Alert.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["routing_key"]).To(Equal("rk-1"))
			Expect(data["auto_start"]).To(BeTrue())
		})
	})
})
