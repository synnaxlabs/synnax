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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration v0", func() {
	It("Should read a status whose task key was stored as float64", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
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
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
			Search:       searchIdx,
		}))

		testRack := &rack.Rack{Name: "Migration Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

		taskKey := task.NewKey(testRack.Key, 99)
		t := task.Task{
			Key:  taskKey,
			Name: "Legacy Task",
		}
		Expect(gorp.NewCreate[task.Key, task.Task]().
			Entry(&t).
			Exec(ctx, db)).To(Succeed())

		// Write a status using Status[any] with the task key as float64,
		// simulating legacy data where the key was encoded as a msgpack
		// float64 instead of uint64.
		legacyStatus := status.Status[any]{
			Key:     task.OntologyID(taskKey).String(),
			Name:    "Legacy Task",
			Variant: xstatus.VariantSuccess,
			Message: "Started",
			Time:    telem.Now(),
			Details: map[string]any{
				"task":    float64(taskKey),
				"running": true,
				"cmd":     "start",
			},
		}
		Expect(status.NewWriter[any](stat, nil).Set(ctx, &legacyStatus)).To(Succeed())

		// Opening the task service triggers the v0.status_backfill migration,
		// which reads existing statuses as Status[StatusDetails]. This would
		// fail without the flex DecodeMsgpack on the Key type because the
		// task key is stored as a msgpack float64.
		MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
			Status:   stat,
			Search:   searchIdx,
		}))

		// Verify the status is readable with the correct typed key.
		var restoredStatus task.Status
		Expect(status.NewRetrieve[task.StatusDetails](stat).
			WhereKeys(task.OntologyID(taskKey).String()).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Details.Task).To(Equal(taskKey))
		Expect(restoredStatus.Details.Running).To(BeTrue())
	})
})
