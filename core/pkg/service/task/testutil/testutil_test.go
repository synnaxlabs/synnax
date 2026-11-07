// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Tasks", func() {
	var (
		db     *gorp.DB
		svc    *arctask.Service
		tasks  *tasktestutil.Tasks
		arcKey arc.Key
	)
	createArc := func(ctx SpecContext) arc.Key {
		GinkgoHelper()
		a := arc.Arc{
			Variant: arc.AutomationArc{Base: arc.Base{
				Key:    uuid.New(),
				Name:   "arc",
				Source: arc.Source{Variant: arc.TextSource{}},
			}},
		}
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		return a.Base().Key
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db, Ontology: otg}))
		arcKey = createArc(ctx)
		svc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:        db,
			Stores:    svc.Stores(),
			MatchArcs: svc.MatchArcs,
		}))
	})

	It("Should create a task whose config lives in the store", func(ctx SpecContext) {
		key := MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key": arcKey.String(),
		}))
		data := MustSucceed(svc.Config.Read(ctx, nil, key))
		Expect(data["arc_key"]).To(Equal(arcKey.String()))
		Expect(data["performance"]).To(Equal("auto"))
	})

	It("Should return the error of a config the store rejects", func(ctx SpecContext) {
		Expect(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
			"arc_key":     uuid.New().String(),
			"performance": "BOGUS",
		})).Error().To(MatchError(ContainSubstring("invalid performance: BOGUS")))
	})

	It("Should filter the tasks by Arc", func(ctx SpecContext) {
		create := func(a arc.Key) task.Key {
			GinkgoHelper()
			return MustSucceed(tasks.Create(ctx, svc.Config, msgpack.EncodedJSON{
				"arc_key": a.String(),
			}))
		}
		first, second := create(arcKey), create(arcKey)
		create(createArc(ctx))
		var matched []task.Task
		Expect(tasks.NewRetrieve().
			Where(task.MatchArcs(arcKey)).
			Entries(&matched).
			Exec(ctx, db)).To(Succeed())
		Expect(lo.Map(matched, func(t task.Task, _ int) task.Key { return t.Key })).
			To(ConsistOf(first, second))
	})
})
