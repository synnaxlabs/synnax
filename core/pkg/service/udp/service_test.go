// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package udp_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/udp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		svc   *udp.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(udp.OpenService(ctx, udp.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(udp.OpenService(ctx, udp.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per UDP task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"udp_read",
				"udp_write",
			))
		})
	})

	Describe("Write", func() {
		It("Should stamp the ICD hash into a read config", func(ctx SpecContext) {
			ic, m := createICD(ctx)
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"icd": ic.Key.String(),
				"messages": []any{map[string]any{
					"message": m.Key.String(),
					"index":   1,
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["icd_hash"]).To(Equal(MustSucceed(
				icd.TaskHash(ic, []icd.MessageKey{m.Key}),
			)))
		})

		It("Should reject a read config whose message is not in the ICD", func(
			ctx SpecContext,
		) {
			ic, _ := createICD(ctx)
			Expect(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"icd":      ic.Key.String(),
				"messages": []any{map[string]any{"message": uuid.New().String()}},
			})).Error().To(MatchError(ContainSubstring("is not in the ICD")))
		})

		It("Should reject a write config whose ICD does not exist", func(
			ctx SpecContext,
		) {
			Expect(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"icd": uuid.New().String(),
			})).Error().To(MatchError(ContainSubstring("does not exist")))
		})

		DescribeTable("Should reject a read config whose rate is not positive",
			func(ctx SpecContext, rate float64) {
				ic, _ := createICD(ctx)
				Expect(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
					"icd":  ic.Key.String(),
					"rate": rate,
				})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
			},
			Entry("zero", 0.0),
			Entry("negative", -1.0),
		)
	})
})
