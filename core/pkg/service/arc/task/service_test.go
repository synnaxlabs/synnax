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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/arc/task/versions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	var (
		db     *gorp.DB
		svc    *arctask.Service
		tasks  *tasktestutil.Tasks
		arcKey arc.Key
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := arc.Arc{
			Variant: arc.AutomationArc{Base: arc.Base{
				Key:    uuid.New(),
				Name:   "arc",
				Source: arc.Source{Variant: arc.TextSource{}},
			}},
		}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		arcKey = a.Base().Key
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
			DB: db,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:        db,
			Stores:    svc.Stores(),
			MatchArcs: svc.MatchArcs,
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

	Describe("MatchArcs", func() {
		It(
			"Should match only the tasks that run the given Arcs",
			func(ctx SpecContext) {
				other := arc.Arc{
					Variant: arc.AutomationArc{Base: arc.Base{
						Key:    uuid.New(),
						Name:   "other",
						Source: arc.Source{Variant: arc.TextSource{}},
					}},
				}
				Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&other).Exec(ctx, db)).
					To(Succeed())
				create := func(a arc.Key) task.Key {
					GinkgoHelper()
					return MustSucceed(
						tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
							"arc_key": a.String(),
						}),
					)
				}
				first, second := create(arcKey), create(arcKey)
				create(other.Base().Key)
				var matched []task.Task
				Expect(tasks.NewRetrieve().
					Where(svc.MatchArcs(arcKey)).
					Entries(&matched).
					Exec(ctx, db)).To(Succeed())
				Expect(
					lo.Map(matched, func(t task.Task, _ int) task.Key { return t.Key }),
				).
					To(ConsistOf(first, second))
			},
		)
	})

	Describe("Write", func() {
		It("Should store a decoded config under the given key", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
				"arc_key":     arcKey.String(),
				"performance": "high",
			}))
			data := MustSucceed(svc.Config.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["arc_key"]).To(Equal(arcKey.String()))
			Expect(data["performance"]).To(Equal("high"))
		})

		It("Should apply schema defaults to absent fields", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
				"arc_key": arcKey.String(),
			}))
			data := MustSucceed(svc.Config.Read(ctx, db, key))
			Expect(data["performance"]).To(Equal("auto"))
		})

		It("Should reject a config whose Arc does not exist", func(ctx SpecContext) {
			missing := uuid.New()
			Expect(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
				"arc_key": missing.String(),
			})).Error().To(MatchError(ContainSubstring(
				"arc_key: arc " + missing.String() + " does not exist",
			)))
		})

		It(
			"Should return the config's validation error for an invalid performance",
			func(
				ctx SpecContext,
			) {
				Expect(svc.Config.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"arc_key":     uuid.New().String(),
					"performance": "BOGUS",
				})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
			},
		)
	})

	Describe("Normalize", func() {
		It(
			"Should convert the loop mode of a v1 config to a performance level",
			func(ctx SpecContext) {
				data := MustSucceed(
					svc.Config.Normalize(ctx, db, 1, msgpack.EncodedJSON{
						"arc_key":        arcKey.String(),
						"execution_mode": "BUSY_WAIT",
						"rt_priority":    80,
					}),
				)
				Expect(data).To(HaveKeyWithValue("performance", "high"))
				Expect(data).ToNot(HaveKey("execution_mode"))
			},
		)

		It("Should decode a current config without rewriting it", func(
			ctx SpecContext,
		) {
			data := MustSucceed(svc.Config.Normalize(ctx, db,
				versions.Latest,
				msgpack.EncodedJSON{
					"arc_key":        arcKey.String(),
					"performance":    "low",
					"execution_mode": "BUSY_WAIT",
				},
			))
			Expect(data).To(HaveKeyWithValue("performance", "low"))
			Expect(data).ToNot(HaveKey("execution_mode"))
		})
	})
})
