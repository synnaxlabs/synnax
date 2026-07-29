// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var (
		db        *gorp.DB
		statusSvc *status.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
	})

	runMigration := func(ctx context.Context) {
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Task",
			Migrations: v0.NewMigrations(v0.MigrationConfig{Status: statusSvc}),
		})).To(Succeed())
	}

	It("Should create unknown statuses for tasks missing them", func(ctx SpecContext) {
		t := v0.Task{
			Key:  v0.Key(65537<<32 | 99),
			Name: "Migration Test Task",
		}
		Expect(gorp.NewCreate[v0.Key, v0.Task]().
			Entry(&t).
			Exec(ctx, db)).To(Succeed())

		runMigration(ctx)

		var restoredStatus status.Status[v0.StatusDetails]
		Expect(status.NewRetrieve[v0.StatusDetails](statusSvc).
			Where(status.MatchKeys[v0.StatusDetails](t.OntologyID().String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Migration Test Task status unknown"))
		Expect(restoredStatus.Details.Task).To(Equal(t.Key))
	})

	It(
		"Should not create statuses for tasks that already have them",
		func(ctx SpecContext) {
			t := v0.Task{
				Key:  v0.Key(65537<<32 | 1),
				Name: "Task With Status",
			}
			Expect(gorp.NewCreate[v0.Key, v0.Task]().
				Entry(&t).
				Exec(ctx, db)).To(Succeed())
			existing := status.Status[v0.StatusDetails]{
				Key:     t.OntologyID().String(),
				Name:    t.Name,
				Variant: status.VariantSuccess,
				Message: "Started",
				Time:    telem.Now(),
				Details: v0.StatusDetails{Task: t.Key},
			}
			Expect(status.NewWriter[v0.StatusDetails](statusSvc, nil).
				Set(ctx, &existing)).To(Succeed())

			runMigration(ctx)

			var taskStatus status.Status[v0.StatusDetails]
			Expect(status.NewRetrieve[v0.StatusDetails](statusSvc).
				Where(status.MatchKeys[v0.StatusDetails](t.OntologyID().String())).
				Entry(&taskStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(taskStatus.Variant).To(Equal(status.VariantSuccess))
			Expect(taskStatus.Message).To(Equal("Started"))
		},
	)
})

var _ = Describe("Status backfill", func() {
	It(
		"Should read a status whose task key was stored as float64",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Search:   searchIdx,
			}))
			statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
				Ontology: otg,
				DB:       db,
				Group:    groupSvc,
				Label:    labelSvc,
				Search:   searchIdx,
			}))

			taskKey := v0.Key(65537<<32 | 99)
			t := v0.Task{
				Key:  taskKey,
				Name: "Legacy Task",
			}
			Expect(gorp.NewCreate[v0.Key, v0.Task]().
				Entry(&t).
				Exec(ctx, db)).To(Succeed())

			// Write a status using Status[any] with the task key as float64, simulating
			// legacy data where the key was encoded as a MessagePack float64 instead of
			// uint64.
			legacyStatus := status.Status[any]{
				Key:     taskKey.OntologyID().String(),
				Name:    "Legacy Task",
				Variant: status.VariantSuccess,
				Message: "Started",
				Time:    telem.Now(),
				Details: map[string]any{
					"task":    float64(taskKey),
					"running": true,
					"cmd":     "start",
				},
			}
			Expect(
				status.NewWriter[any](statusSvc, nil).Set(ctx, &legacyStatus),
			).To(Succeed())

			// The backfill reads existing statuses as Status[StatusDetails]. This would
			// fail without the flex DecodeMsgpack on the Key type because the task key
			// is stored as a MessagePack float64.
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Task",
				Migrations: v0.NewMigrations(v0.MigrationConfig{Status: statusSvc}),
			})).To(Succeed())

			// Verify the status is readable with the correct typed key.
			var restoredStatus status.Status[v0.StatusDetails]
			Expect(status.NewRetrieve[v0.StatusDetails](statusSvc).
				Where(status.MatchKeys[v0.StatusDetails](taskKey.OntologyID().String())).
				Entry(&restoredStatus).
				Exec(ctx, nil)).To(Succeed())
			Expect(restoredStatus.Details.Task).To(Equal(taskKey))
			Expect(restoredStatus.Details.Running).To(BeTrue())
		},
	)
})
