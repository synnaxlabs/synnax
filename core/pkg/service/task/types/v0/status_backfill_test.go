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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/types/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Status backfill", func() {
	It("Should read a status whose task key was stored as float64", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
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
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
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

		// Write a status using Status[any] with the task key as float64,
		// simulating legacy data where the key was encoded as a msgpack
		// float64 instead of uint64.
		legacyStatus := status.Status[any]{
			Key:     v0.OntologyID(taskKey).String(),
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
		Expect(status.NewWriter[any](stat, nil).Set(ctx, &legacyStatus)).To(Succeed())

		// The backfill reads existing statuses as Status[StatusDetails]. This
		// would fail without the flex DecodeMsgpack on the Key type because
		// the task key is stored as a msgpack float64.
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Task",
			Migrations: []migrate.Migration{
				v0.NewMigration(v0.MigrationConfig{Status: stat}),
			},
		})).To(Succeed())

		// Verify the status is readable with the correct typed key.
		var restoredStatus status.Status[v0.StatusDetails]
		Expect(status.NewRetrieve[v0.StatusDetails](stat).
			Where(status.MatchKeys[v0.StatusDetails](v0.OntologyID(taskKey).String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Details.Task).To(Equal(taskKey))
		Expect(restoredStatus.Details.Running).To(BeTrue())
	})
})
