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
	"strconv"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	v56 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v56"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrations", func() {
	It("Should migrate a legacy uint64-keyed task and its status to a UUID key", func(ctx SpecContext) {
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
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))

		testRack := &rack.Rack{Name: "Migration Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

		legacyKey := v56.Key(uint64(testRack.Key)<<32 | 99)
		legacyTask := v56.Task{Key: legacyKey, Name: "Legacy Task"}
		Expect(gorp.NewCreate[v56.Key, v56.Task]().
			Entry(&legacyTask).
			Exec(ctx, db)).To(Succeed())
		legacyID := ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  strconv.FormatUint(uint64(legacyKey), 10),
		}
		Expect(otg.NewWriter(nil).DefineResources(ctx, legacyID)).To(Succeed())

		// Legacy data stored the task key as a msgpack float64; the v0 backfill
		// must read it through the flex decoder before the re-key runs.
		legacyStatus := status.Status[any]{
			Key:     "task:" + strconv.FormatUint(uint64(legacyKey), 10),
			Name:    "Legacy Task",
			Variant: status.VariantSuccess,
			Message: "Started",
			Time:    telem.Now(),
			Details: map[string]any{
				"task":    float64(legacyKey),
				"running": true,
				"cmd":     "start",
			},
		}
		Expect(status.NewWriter[any](stat, nil).Set(ctx, &legacyStatus)).To(Succeed())

		svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
			Status:   stat,
			Search:   searchIdx,
		}))

		var migrated task.Task
		Expect(svc.NewRetrieve().
			Where(task.MatchNames("Legacy Task")).
			Entry(&migrated).
			Exec(ctx, nil)).To(Succeed())
		Expect(migrated.Key).ToNot(Equal(uuid.Nil))
		Expect(migrated.Rack).To(Equal(testRack.Key))

		var restoredStatus task.Status
		Expect(status.NewRetrieve[task.StatusDetails](stat).
			Where(status.MatchKeys[task.StatusDetails](task.OntologyID(migrated.Key).String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Details.Task).To(Equal(migrated.Key))
		Expect(restoredStatus.Details.Running).To(BeTrue())
		Expect(restoredStatus.Details.Cmd).To(Equal("start"))

		Expect(otg.NewRetrieve().
			WhereIDs(task.OntologyID(migrated.Key)).
			Exists(ctx, nil)).To(BeTrue())
		Expect(otg.NewRetrieve().WhereIDs(legacyID).Exists(ctx, nil)).To(BeFalse())

		staged, closer := MustSucceed2(db.Get(ctx, task.LegacyKeyKVKey(legacyKey)))
		Expect(string(staged)).To(Equal(migrated.Key.String()))
		Expect(closer.Close()).To(Succeed())
	})
})
