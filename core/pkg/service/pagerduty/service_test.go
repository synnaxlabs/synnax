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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-1",
			})).To(Succeed())
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-2",
			})).To(Succeed())
			data := MustSucceed(svc.Alert.Read(ctx, nil, key))
			Expect(data["routing_key"]).To(Equal("rk-2"))
		})

		It("Should return a validation error for a malformed config", func(
			ctx SpecContext,
		) {
			Expect(svc.Alert.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"alerts": "not-an-array",
			})).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Alert.Read(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-1",
			})).To(Succeed())
			Expect(svc.Alert.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.Alert.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Alert.Delete(ctx, nil, key)).To(Succeed())
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			from, to := uuid.New(), uuid.New()
			Expect(svc.Alert.Write(ctx, nil, from, msgpack.EncodedJSON{
				"routing_key": "rk-1",
			})).To(Succeed())
			Expect(svc.Alert.Copy(ctx, nil, from, to)).To(Succeed())
			data := MustSucceed(svc.Alert.Read(ctx, nil, to))
			Expect(data["key"]).To(Equal(to.String()))
			Expect(data["routing_key"]).To(Equal("rk-1"))
			original := MustSucceed(svc.Alert.Read(ctx, nil, from))
			Expect(original["key"]).To(Equal(from.String()))
		})

		It("Should return not found when the source is missing", func(
			ctx SpecContext,
		) {
			Expect(svc.Alert.Copy(ctx, nil, uuid.New(), uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Ontology", func() {
		It("Should serve stored records as resources", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-1",
			})).To(Succeed())
			res := MustSucceed(svc.Alert.RetrieveResource(ctx, key.String(), nil))
			Expect(res.ID).To(Equal(ontology.ID{
				Type: ontology.ResourceTypePagerdutyAlert,
				Key:  key.String(),
			}))
		})

		It("Should resolve resources through the registered ontology", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Alert.Write(ctx, nil, key, msgpack.EncodedJSON{
				"routing_key": "rk-1",
			})).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ontology.ID{
				Type: ontology.ResourceTypePagerdutyAlert,
				Key:  key.String(),
			}).Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal("pagerduty_alert"))
		})
	})
})
