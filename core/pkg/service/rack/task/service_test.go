// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var svc *task.Service
	BeforeEach(func(ctx SpecContext) {
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	Describe("Stores", func() {
		It("Should expose the rack status store", func() {
			types := []ontology.ResourceType{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(ontology.ResourceTypeRackStatus))
		})
	})

	Describe("Write", func() {
		It("Should store an empty config under the given key", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Status.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Status.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
		})

		It("Should return a validation error for a malformed config", func(
			ctx SpecContext,
		) {
			Expect(svc.Status.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"key": 123,
			})).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.Status.Read(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Status.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			Expect(svc.Status.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.Status.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.Status.Delete(ctx, nil, key)).To(Succeed())
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			from, to := uuid.New(), uuid.New()
			Expect(
				svc.Status.Write(ctx, nil, from, msgpack.EncodedJSON{}),
			).To(Succeed())
			Expect(svc.Status.Copy(ctx, nil, from, to)).To(Succeed())
			data := MustSucceed(svc.Status.Read(ctx, nil, to))
			Expect(data["key"]).To(Equal(to.String()))
			original := MustSucceed(svc.Status.Read(ctx, nil, from))
			Expect(original["key"]).To(Equal(from.String()))
		})
	})

	Describe("Ontology", func() {
		It("Should serve stored records as resources", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Status.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			res := MustSucceed(svc.Status.RetrieveResource(ctx, key.String(), nil))
			Expect(res.ID).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeRackStatus,
				Key:  key.String(),
			}))
		})
	})
})
