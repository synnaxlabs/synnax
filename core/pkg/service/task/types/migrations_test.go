// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/types"
	taskv0 "github.com/synnaxlabs/synnax/pkg/service/task/types/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db   *gorp.DB
		stat *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
	})

	runMigrations := func(ctx SpecContext) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Task",
			Migrations: types.NewMigrations(types.MigrationsConfig{Status: stat}),
		})).To(Succeed())
	}

	It("Should create unknown statuses for tasks missing them", func(ctx SpecContext) {
		t := taskv0.Task{
			Key:  taskv0.Key(65537<<32 | 99),
			Name: "Migration Test Task",
		}
		Expect(gorp.NewCreate[taskv0.Key, taskv0.Task]().
			Entry(&t).
			Exec(ctx, db)).To(Succeed())

		runMigrations(ctx)

		var restoredStatus types.Status
		Expect(status.NewRetrieve[types.StatusDetails](stat).
			Where(status.MatchKeys[types.StatusDetails](taskv0.OntologyID(t.Key).String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Migration Test Task status unknown"))
		Expect(restoredStatus.Details.Task).To(Equal(t.Key))
	})

	It("Should not create statuses for tasks that already have them", func(ctx SpecContext) {
		t := taskv0.Task{
			Key:  taskv0.Key(65537<<32 | 1),
			Name: "Task With Status",
		}
		Expect(gorp.NewCreate[taskv0.Key, taskv0.Task]().
			Entry(&t).
			Exec(ctx, db)).To(Succeed())
		existing := types.Status{
			Key:     taskv0.OntologyID(t.Key).String(),
			Name:    t.Name,
			Variant: status.VariantSuccess,
			Message: "Started",
			Time:    telem.Now(),
			Details: types.StatusDetails{Task: t.Key},
		}
		Expect(status.NewWriter[types.StatusDetails](stat, nil).
			Set(ctx, &existing)).To(Succeed())

		runMigrations(ctx)

		var taskStatus types.Status
		Expect(status.NewRetrieve[types.StatusDetails](stat).
			Where(status.MatchKeys[types.StatusDetails](taskv0.OntologyID(t.Key).String())).
			Entry(&taskStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(taskStatus.Variant).To(Equal(status.VariantSuccess))
		Expect(taskStatus.Message).To(Equal("Started"))
	})
})
