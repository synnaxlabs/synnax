// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("RangeMigration", func() {
	var (
		db       *gorp.DB
		rng      = uuid.New()
		gone     = uuid.New()
		inRange  = uuid.New()
		inGone   = uuid.New()
		inArc    = uuid.New()
		onRack   = uuid.New()
		rackID   = resource.ID{Type: resource.TypeRack, Key: "65537"}
		toRack   ontology.Relationship
		tasks    map[v3.Key]v3.Task
		retained []ontology.Relationship
	)
	parentOf := func(from resource.ID, to uuid.UUID) ontology.Relationship {
		return ontology.Relationship{
			From: from,
			Type: ontology.RelationshipTypeParentOf,
			To:   resource.ID{Type: resource.TypeTask, Key: to.String()},
		}
	}
	rangeID := func(k uuid.UUID) resource.ID {
		return resource.ID{Type: resource.TypeRange, Key: k.String()}
	}

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		stored := []v2.Task{
			{Key: inRange, Name: "In Range", Snapshot: true},
			{Key: inGone, Name: "In Deleted Range", Snapshot: true},
			{Key: inArc, Name: "Arc Task"},
			{Key: onRack, Name: "Rack Task"},
		}
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entries(&stored).Exec(ctx, db)).
			To(Succeed())
		resources := []ontology.Resource{{ID: rangeID(rng)}}
		Expect(gorp.NewCreate[string, ontology.Resource]().
			Entries(&resources).
			Exec(ctx, db)).To(Succeed())
		toRack = parentOf(rackID, onRack)
		edges := []ontology.Relationship{
			parentOf(rangeID(rng), inRange),
			parentOf(rangeID(gone), inGone),
			parentOf(
				resource.ID{Type: resource.TypeArc, Key: uuid.New().String()},
				inArc,
			),
			toRack,
		}
		Expect(gorp.NewCreate[string, ontology.Relationship]().
			Entries(&edges).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Task",
			Migrations: []gorp.Migration{v3.Migration, v3.RangeMigration},
		})).To(Succeed())
		var out []v3.Task
		Expect(gorp.NewRetrieve[v3.Key, v3.Task]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		tasks = make(map[v3.Key]v3.Task, len(out))
		for _, t := range out {
			tasks[t.Key] = t
		}
		retained = nil
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&retained).
			Exec(ctx, db)).To(Succeed())
	})

	It("Should keep every task", func() {
		Expect(tasks).To(HaveLen(4))
		Expect(tasks[inRange].Name).To(Equal("In Range"))
	})

	It("Should set the range of a task in a stored range", func() {
		Expect(tasks[inRange].Range).To(HaveValue(Equal(rng)))
	})

	DescribeTable("Should leave no range on a task",
		func(key func() uuid.UUID) { Expect(tasks[key()].Range).To(BeNil()) },
		Entry("in a deleted range", func() uuid.UUID { return inGone }),
		Entry("under an Arc", func() uuid.UUID { return inArc }),
		Entry("on a rack", func() uuid.UUID { return onRack }),
	)

	It("Should delete every edge from a range or an Arc to a task", func() {
		Expect(retained).To(ConsistOf(toRack))
	})
})
