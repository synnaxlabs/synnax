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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskv0 "github.com/synnaxlabs/synnax/pkg/service/task/types/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	It("Should create unknown statuses for tasks missing them", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
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

		t := taskv0.Task{
			Key:  taskv0.Key(task.NewKey(testRack.Key, 99)),
			Name: "Migration Test Task",
		}
		Expect(gorp.NewCreate[taskv0.Key, taskv0.Task]().
			Entry(&t).
			Exec(ctx, db)).To(Succeed())

		MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
			Status:   stat,
			Search:   searchIdx,
		}))

		var restoredStatus task.Status
		Expect(status.NewRetrieve[task.StatusDetails](stat).
			Where(status.MatchKeys[task.StatusDetails](task.OntologyID(task.Key(t.Key)).String())).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Variant).To(Equal(status.VariantWarning))
		Expect(restoredStatus.Message).To(Equal("Migration Test Task status unknown"))
		Expect(restoredStatus.Details.Task).To(Equal(task.Key(t.Key)))
	})

	It("Should not create statuses for tasks that already have them", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
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

		svc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
			Status:   stat,
			Search:   searchIdx,
		}))
		t := &task.Task{
			Key:  task.NewKey(testRack.Key, 0),
			Name: "Task With Status",
		}
		Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

		var taskStatus task.Status
		Expect(status.NewRetrieve[task.StatusDetails](stat).
			Where(status.MatchKeys[task.StatusDetails](task.OntologyID(t.Key).String())).
			Entry(&taskStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(taskStatus.Variant).To(Equal(status.VariantWarning))
		Expect(taskStatus.Message).To(Equal("Task With Status status unknown"))
	})
})
