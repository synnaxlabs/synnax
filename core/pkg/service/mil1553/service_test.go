// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mil1553_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/bus"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/mil1553"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *mil1553.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(mil1553.OpenService(ctx, mil1553.ServiceConfig{
			DB: db,
			Resolver: bus.Resolver{
				Stamper: library.Stamper{DB: db, Ontology: otg},
			},
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the resolver", func(ctx SpecContext) {
			Expect(mil1553.OpenService(ctx, mil1553.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring(
					"resolver.stamper.db: must be non-nil",
				)))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per MIL-STD-1553 task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"mil1553_read",
				"mil1553_write",
				"mil1553_scan",
			))
		})
	})

	Describe("Write", func() {
		It("Should stamp the library hash into a read config", func(ctx SpecContext) {
			key := uuid.New()
			l, m := createLibrary(ctx, key)
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"library": l.Key.String(),
				"messages": []any{map[string]any{
					"message": m.Key.String(),
					"index":   1,
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["library_hash"]).To(Equal(MustSucceed(library.Hash(l))))
		})

		It("Should reject a read config whose message is not in the library", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			l, _ := createLibrary(ctx, key)
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"library":  l.Key.String(),
				"messages": []any{map[string]any{"message": uuid.New().String()}},
			})).To(MatchError(ContainSubstring("is not in the library")))
		})

		It("Should reject a write config whose library does not exist", func(
			ctx SpecContext,
		) {
			Expect(svc.Write.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"library": uuid.New().String(),
			})).To(MatchError(ContainSubstring("does not exist")))
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
