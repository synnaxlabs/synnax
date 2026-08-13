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
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *arctask.Service
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(arctask.OpenService(ctx, arctask.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose the Arc task store", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf("arc"))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded config under the given key", func(
			ctx SpecContext,
		) {
			key, arcKey := uuid.New(), uuid.New()
			Expect(svc.Config.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key":        arcKey.String(),
				"execution_mode": "AUTO",
				"rt_priority":    10,
			})).To(Succeed())
			data := MustSucceed(svc.Config.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["execution_mode"]).To(Equal("AUTO"))
			Expect(data["rt_priority"]).To(BeNumerically("==", 10))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Config.Write(ctx, nil, key, msgpack.EncodedJSON{
				"arc_key": uuid.New().String(),
			})).To(Succeed())
			data := MustSucceed(svc.Config.Read(ctx, nil, key))
			Expect(data["execution_mode"]).To(Equal("AUTO"))
			Expect(data["rt_priority"]).To(BeNumerically("==", 47))
			Expect(data["cpu_affinity"]).To(BeNumerically("==", -1))
		})

		It("Should return the config's validation error for an invalid mode", func(
			ctx SpecContext,
		) {
			Expect(svc.Config.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"arc_key":        uuid.New().String(),
				"execution_mode": "BOGUS",
			})).To(MatchError(ContainSubstring("invalid execution_mode: BOGUS")))
		})
	})
})
