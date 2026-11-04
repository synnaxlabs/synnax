// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	log "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var rootID = resource.ID{Type: resource.TypeBuiltin, Key: "root"}

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		Type: ontology.RelationshipTypeParentOf,
		To:   to,
	}
}

func createEdges(ctx context.Context, db *gorp.DB, edges ...ontology.Relationship) {
	GinkgoHelper()
	Expect(gorp.NewCreate[string, ontology.Relationship]().
		Entries(&edges).
		Exec(ctx, db)).To(Succeed())
}

func edges(ctx context.Context, db *gorp.DB) []ontology.Relationship {
	GinkgoHelper()
	var out []ontology.Relationship
	Expect(gorp.NewRetrieve[string, ontology.Relationship]().
		Entries(&out).
		Exec(ctx, db)).To(Succeed())
	return out
}

func groupID(k uuid.UUID) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: k.String()}
}

var _ = Describe("ParentMigration", func() {
	var (
		db       *gorp.DB
		channels = uuid.New()
		users    = uuid.New()
		a        = uuid.New()
		b        = uuid.New()
		inUsers  = uuid.New()
		ranges   = uuid.New()
		inRanges = uuid.New()
		inProj   = uuid.New()
		inRack   = uuid.New()
		nested   = uuid.New()
		orphan   = uuid.New()
		project  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		rack     = resource.ID{Type: resource.TypeRack, Key: "65537"}
		toProj   = parentOf(rootID, project)
		toRanges = parentOf(rootID, groupID(ranges))
		inRange  = parentOf(groupID(ranges), groupID(inRanges))
		groups   map[uuid.UUID]v1.Group
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		names := map[uuid.UUID]string{
			channels: "Channels", users: "Users", a: "A", b: "B", inUsers: "In Users",
			inProj: "In Project", inRack: "In Rack", nested: "Nested", orphan: "Orphan",
			ranges: "Ranges", inRanges: "In Ranges",
		}
		for k, name := range names {
			g := v0.Group{Key: k, Name: name}
			Expect(gorp.NewCreate[v0.Key, v0.Group]().Entry(&g).Exec(ctx, db)).
				To(Succeed())
		}
		createEdges(ctx, db,
			parentOf(rootID, groupID(channels)),
			parentOf(rootID, groupID(users)),
			parentOf(groupID(channels), groupID(a)),
			parentOf(groupID(a), groupID(b)),
			parentOf(groupID(users), groupID(inUsers)),
			parentOf(project, groupID(inProj)),
			parentOf(rack, groupID(inRack)),
			parentOf(groupID(inProj), groupID(nested)),
			toProj,
			toRanges,
			inRange,
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "group",
			Migrations: []gorp.Migration{v1.Migration, v1.ParentMigration},
		})).To(Succeed())
		var out []v1.Group
		Expect(gorp.NewRetrieve[v1.Key, v1.Group]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		groups = make(map[uuid.UUID]v1.Group, len(out))
		for _, g := range out {
			groups[g.Key] = g
		}
	})

	DescribeTable("Should derive the scope and parent of each group",
		func(key *uuid.UUID, scope resource.Type, parent func() *resource.ID) {
			g := groups[*key]
			Expect(g.Scope).To(Equal(scope))
			Expect(g.Parent).To(Equal(parent()))
		},
		Entry("a built-in root", &channels, resource.TypeBuiltin, func() *resource.ID {
			return nil
		}),
		Entry("a group under a built-in root", &a, resource.TypeChannel,
			func() *resource.ID { return nil }),
		Entry("a group under a group", &b, resource.TypeChannel,
			func() *resource.ID { return new(groupID(a)) }),
		Entry("a group under a root with no scope", &inUsers, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
		Entry("a group under a project", &inProj, resource.TypeProject,
			func() *resource.ID { return &project }),
		Entry("a group under a rack", &inRack, resource.TypeDevice,
			func() *resource.ID { return &rack }),
		Entry("a group under a project group", &nested, resource.TypeProject,
			func() *resource.ID { return new(groupID(inProj)) }),
		Entry("a group with no parent edge", &orphan, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
	)

	It(
		"Should delete the parent edges into groups and keep the rest",
		func(ctx SpecContext) {
			Expect(edges(ctx, db)).To(ConsistOf(toProj, toRanges, inRange))
		},
	)
})

var _ = Describe("MembersMigration", func() {
	var (
		db       *gorp.DB
		root     = v1.Group{Key: uuid.New(), Name: "Logs", Scope: resource.TypeBuiltin}
		top      = v1.Group{Key: uuid.New(), Name: "Top", Scope: resource.TypeProject}
		project  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		projTop  = v1.Group{Key: uuid.New(), Name: "P", Scope: resource.TypeProject}
		projLeaf = v1.Group{Key: uuid.New(), Name: "Leaf", Scope: resource.TypeProject}
		logs     = []log.Log{
			{Key: uuid.New(), Name: "In root"},
			{Key: uuid.New(), Name: "In top"},
			{Key: uuid.New(), Name: "In project"},
			{Key: uuid.New(), Name: "Owned"},
		}
		logID = func(i int) resource.ID {
			return resource.ID{Type: resource.TypeLog, Key: logs[i].Key.String()}
		}
		migrate = func(ctx context.Context, projectOwned bool) []log.Log {
			GinkgoHelper()
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "log",
				Migrations: []gorp.Migration{
					v1.NewMembersMigration(v1.MembersConfig[log.Key, log.Log]{
						Name:         "members",
						Type:         resource.TypeLog,
						SetGroup:     func(l *log.Log, g v1.Key) { l.Group = &g },
						ProjectOwned: projectOwned,
					}),
				},
			})).To(Succeed())
			out := make([]log.Log, len(logs))
			for i, l := range logs {
				Expect(gorp.NewRetrieve[log.Key, log.Log]().
					Where(gorp.MatchKeys[log.Key, log.Log](l.Key)).
					Entry(&out[i]).
					Exec(ctx, db)).To(Succeed())
			}
			return out
		}
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		projTop.Parent = &project
		projLeaf.Parent = new(groupID(projTop.Key))
		groups := []v1.Group{root, top, projTop, projLeaf}
		Expect(gorp.NewCreate[v1.Key, v1.Group]().Entries(&groups).Exec(ctx, db)).
			To(Succeed())
		created := append([]log.Log(nil), logs...)
		Expect(gorp.NewCreate[log.Key, log.Log]().Entries(&created).Exec(ctx, db)).
			To(Succeed())
		createEdges(ctx, db,
			parentOf(groupID(root.Key), logID(0)),
			parentOf(groupID(top.Key), logID(1)),
			parentOf(groupID(projLeaf.Key), logID(2)),
			parentOf(project, logID(3)),
		)
	})

	It("Should set the group of a member under a scoped group", func(ctx SpecContext) {
		out := migrate(ctx, false)
		Expect(out[1].Group).To(Equal(&top.Key))
		Expect(out[2].Group).To(Equal(&projLeaf.Key))
	})

	It(
		"Should leave a member under a built-in root with no group",
		func(ctx SpecContext) {
			out := migrate(ctx, false)
			Expect(out[0].Group).To(BeNil())
			Expect(out[3].Group).To(BeNil())
		},
	)

	It("Should delete every edge from a group to a member", func(ctx SpecContext) {
		migrate(ctx, false)
		Expect(edges(ctx, db)).To(ConsistOf(parentOf(project, logID(3))))
	})

	It(
		"Should add an edge from the project that owns a grouped member",
		func(ctx SpecContext) {
			migrate(ctx, true)
			Expect(edges(ctx, db)).To(ConsistOf(
				parentOf(project, logID(3)),
				parentOf(project, logID(2)),
			))
		},
	)
})
