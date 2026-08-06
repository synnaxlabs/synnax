// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ni_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		otg *ontology.Ontology
		svc *ni.Service
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		svc = MustOpen(ni.OpenService(ctx, ni.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})

	Describe("Stores", func() {
		It("Should expose one store per NI task type", func() {
			types := []ontology.ResourceType{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				ontology.ResourceTypeNiAnalogRead,
				ontology.ResourceTypeNiAnalogWrite,
				ontology.ResourceTypeNiCounterRead,
				ontology.ResourceTypeNiDigitalRead,
				ontology.ResourceTypeNiDigitalWrite,
				ontology.ResourceTypeNiScanner,
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded analog read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
				"sample_rate": 25,
				"stream_rate": 5,
				"channels":    []any{},
			})).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should overwrite the record stored under the same key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(
				svc.Scanner.Write(ctx, nil, key, msgpack.EncodedJSON{}),
			).To(Succeed())
			Expect(
				svc.Scanner.Write(ctx, nil, key, msgpack.EncodedJSON{}),
			).To(Succeed())
			data := MustSucceed(svc.Scanner.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
		})

		It("Should return a validation error for a malformed config", func(
			ctx SpecContext,
		) {
			Expect(svc.AnalogRead.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"channels": "not-an-array",
			})).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Read", func() {
		It("Should return not found for a missing record", func(ctx SpecContext) {
			Expect(svc.CounterRead.Read(ctx, nil, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove a stored record idempotently", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.DigitalWrite.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device":     "dev-1",
				"state_rate": 10,
			})).To(Succeed())
			Expect(svc.DigitalWrite.Delete(ctx, nil, key)).To(Succeed())
			Expect(svc.DigitalWrite.Read(ctx, nil, key)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(svc.DigitalWrite.Delete(ctx, nil, key)).To(Succeed())
		})
	})

	Describe("Copy", func() {
		It("Should duplicate a record under a new key", func(ctx SpecContext) {
			from, to := uuid.New(), uuid.New()
			Expect(svc.AnalogWrite.Write(ctx, nil, from, msgpack.EncodedJSON{
				"device":     "dev-1",
				"state_rate": 10,
			})).To(Succeed())
			Expect(svc.AnalogWrite.Copy(ctx, nil, from, to)).To(Succeed())
			data := MustSucceed(svc.AnalogWrite.Read(ctx, nil, to))
			Expect(data["key"]).To(Equal(to.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			original := MustSucceed(svc.AnalogWrite.Read(ctx, nil, from))
			Expect(original["key"]).To(Equal(from.String()))
		})

		It("Should return not found when the source is missing", func(
			ctx SpecContext,
		) {
			Expect(svc.DigitalRead.Copy(ctx, nil, uuid.New(), uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Ontology", func() {
		It("Should serve stored records as resources", func(ctx SpecContext) {
			key := uuid.New()
			Expect(
				svc.Scanner.Write(ctx, nil, key, msgpack.EncodedJSON{}),
			).To(Succeed())
			res := MustSucceed(svc.Scanner.RetrieveResource(ctx, key.String(), nil))
			Expect(res.ID).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeNiScanner,
				Key:  key.String(),
			}))
		})

		It("Should resolve resources through the registered ontology", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
				"sample_rate": 25,
				"stream_rate": 5,
			})).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ontology.ID{
				Type: ontology.ResourceTypeNiAnalogRead,
				Key:  key.String(),
			}).Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Name).To(Equal("ni_analog_read"))
		})
	})
})
