// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package common_test

import (
	"context"
	"iter"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// testType is the resource type the suite registers its store under. The record
// type is the Arc task config, picked because it exercises both the ApplyDefaults
// and Validate hooks.
const testType ontology.ResourceType = "arc_task"

var _ = Describe("ConfigService", func() {
	var (
		otg *ontology.Ontology
		svc *common.ConfigService[arctask.Config, *arctask.Config]
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(common.OpenConfigService[arctask.Config](
			ctx,
			common.ConfigServiceConfig{DB: db, Ontology: otg, Type: testType},
		))
	})

	Describe("OpenConfigService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(common.OpenConfigService[arctask.Config](
				ctx, common.ConfigServiceConfig{Ontology: otg, Type: testType},
			)).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the ontology", func(ctx SpecContext) {
			Expect(common.OpenConfigService[arctask.Config](
				ctx, common.ConfigServiceConfig{DB: db, Type: testType},
			)).Error().To(MatchError(ContainSubstring("ontology: must be non-nil")))
		})

		It("Should reject a config missing the type", func(ctx SpecContext) {
			Expect(common.OpenConfigService[arctask.Config](
				ctx, common.ConfigServiceConfig{DB: db, Ontology: otg},
			)).Error().To(MatchError(ContainSubstring("type: required")))
		})
	})

	Describe("Type", func() {
		It("Should report the configured resource type", func() {
			Expect(svc.Type()).To(Equal(testType))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded config under the given key", func(
			ctx SpecContext,
		) {
			key, arcKey := uuid.New(), uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     arcKey.String(),
				"rt_priority": 10,
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["rt_priority"]).To(BeNumerically("==", 10))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["execution_mode"]).To(Equal("AUTO"))
			Expect(data["rt_priority"]).To(BeNumerically("==", 47))
			Expect(data["cpu_affinity"]).To(BeNumerically("==", -1))
		})

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"rt_priority": 10,
			})).To(Succeed())
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":     uuid.New().String(),
				"rt_priority": 20,
			})).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, key))
			Expect(data["rt_priority"]).To(BeNumerically("==", 20))
		})

		It("Should return a validation error when the data does not decode", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key": 123,
			})).To(MatchError(validate.ErrValidation))
		})

		It("Should return the record's validation error when it fails validation", func(
			ctx SpecContext,
		) {
			Expect(svc.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":        uuid.New().String(),
				"execution_mode": "BOGUS",
			})).To(MatchError(ContainSubstring("invalid execution_mode: BOGUS")))
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Read(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
		})

		It("Should remove multiple records in one call", func(ctx SpecContext) {
			k1, k2 := uuid.New(), uuid.New()
			for _, k := range []uuid.UUID{k1, k2} {
				Expect(svc.Write(ctx, nil, k, msgpack.EncodedJSON{
					"arc_key": uuid.New().String(),
				})).To(Succeed())
			}
			Expect(svc.Delete(ctx, nil, k1, k2)).To(Succeed())
			Expect(svc.Read(ctx, nil, k1)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Read(ctx, nil, k2)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			from, to := uuid.New(), uuid.New()
			arcKey := uuid.New()
			Expect(svc.Write(ctx, nil, from, msgpack.EncodedJSON{
				"arc_key": arcKey.String(),
			})).To(Succeed())
			Expect(svc.Copy(ctx, nil, from, to)).To(Succeed())
			data := MustSucceed(svc.Read(ctx, nil, to))
			Expect(data["key"]).To(Equal(to.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			original := MustSucceed(svc.Read(ctx, nil, from))
			Expect(original["key"]).To(Equal(from.String()))
		})

		It("Should return not found when the source is missing", func(
			ctx SpecContext,
		) {
			Expect(svc.Copy(ctx, nil, uuid.New(), uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("RetrieveResource", func() {
		It("Should serve a stored record as a resource", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, key.String(), nil))
			Expect(res.ID).To(Equal(ontology.ID{Type: testType, Key: key.String()}))
		})

		It("Should return an error for a non-UUID key", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, "not-a-uuid", nil)).Error().
				To(MatchError(ContainSubstring("invalid UUID")))
		})

		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, uuid.New().String(), nil)).Error().
				To(MatchError(query.ErrNotFound))
		})

		It("Should resolve resources through the registered ontology", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ontology.ID{
				Type: testType,
				Key:  key.String(),
			}).Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal(string(testType)))
		})
	})

	Describe("OnChange", func() {
		It("Should notify on writes and deletes", func(ctx SpecContext) {
			var (
				mu      sync.Mutex
				changes []ontology.Change
			)
			disconnect := svc.OnChange(
				func(_ context.Context, seq iter.Seq[ontology.Change]) {
					mu.Lock()
					defer mu.Unlock()
					for c := range seq {
						changes = append(changes, c)
					}
				})
			defer disconnect()
			key := uuid.New()
			id := ontology.ID{Type: testType, Key: key.String()}
			Expect(svc.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			last := func() (c ontology.Change) {
				mu.Lock()
				defer mu.Unlock()
				if len(changes) > 0 {
					c = changes[len(changes)-1]
				}
				return c
			}
			Eventually(last).Should(SatisfyAll(
				HaveField("Variant", change.VariantSet),
				HaveField("Key", id.String()),
				HaveField("Value.ID", id),
			))
			Expect(svc.Delete(ctx, nil, key)).To(Succeed())
			Eventually(last).Should(SatisfyAll(
				HaveField("Variant", change.VariantDelete),
				HaveField("Key", id.String()),
			))
		})
	})
})

var _ = Describe("ConfigRegistry", func() {
	var store common.ConfigStore
	BeforeEach(func(ctx SpecContext) {
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		store = MustOpen(common.OpenConfigService[arctask.Config](
			ctx,
			common.ConfigServiceConfig{DB: db, Ontology: otg, Type: testType},
		))
	})

	Describe("NewConfigRegistry", func() {
		It("Should route each store by its type", func() {
			reg := MustSucceed(common.NewConfigRegistry(store))
			Expect(reg.IsZero()).To(BeFalse())
			Expect(MustBeOk(reg.Store(testType))).To(BeIdenticalTo(store))
			Expect(reg.Types()).To(ConsistOf(testType))
		})

		It("Should reject two stores declaring the same type", func() {
			Expect(common.NewConfigRegistry(store, store)).Error().
				To(MatchError(ContainSubstring("registered twice")))
		})
	})

	Describe("Store", func() {
		It("Should return false for an unclaimed type", func() {
			reg := MustSucceed(common.NewConfigRegistry(store))
			_, ok := reg.Store("unclaimed")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("IsZero", func() {
		It("Should report true for a never-constructed registry", func() {
			Expect(common.ConfigRegistry{}.IsZero()).To(BeTrue())
		})
	})
})
