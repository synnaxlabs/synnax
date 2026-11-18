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
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	log "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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
		a        = uuid.MustParse("ffffffff-ffff-4fff-bfff-ffffffffffff")
		b        = uuid.MustParse("00000000-0000-4000-8000-000000000001")
		inUsers  = uuid.New()
		ranges   = uuid.New()
		inRanges = uuid.New()
		inProj   = uuid.New()
		inRack   = uuid.New()
		nested   = uuid.New()
		orphan   = uuid.New()
		implied  = uuid.New()
		inImpl   = uuid.New()
		mixed    = uuid.New()
		unknown  = uuid.New()
		inUnkn   = uuid.New()
		inWs     = uuid.New()
		wsRoot   = uuid.New()
		inWsRoot = uuid.New()
		ofWs     = uuid.New()
		tasks    = uuid.New()
		inTasks  = uuid.New()
		ws       = uuid.New().String()
		channel  = resource.ID{Type: resource.TypeChannel, Key: "1"}
		twoGrps  = resource.ID{Type: resource.TypeChannel, Key: "2"}
		schem    = resource.ID{Type: resource.TypeSchematic, Key: uuid.New().String()}
		rootTask = resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		task     = resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		rng      = resource.ID{Type: resource.TypeRange, Key: uuid.New().String()}
		subProj  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		members  = []ontology.Relationship{
			parentOf(groupID(channels), channel),
			parentOf(groupID(inImpl), channel),
			parentOf(groupID(mixed), channel),
			parentOf(groupID(mixed), schem),
			parentOf(groupID(inUnkn), schem),
			parentOf(groupID(ofWs), resource.ID{Type: "workspace", Key: ws}),
			parentOf(groupID(inProj), channel),
			parentOf(groupID(a), twoGrps),
			parentOf(groupID(b), twoGrps),
			parentOf(groupID(tasks), rootTask),
			parentOf(groupID(inTasks), task),
			parentOf(groupID(inRanges), rng),
			parentOf(groupID(nested), subProj),
			parentOf(groupID(ofWs), subProj),
		}
		project  = resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		rack     = resource.ID{Type: resource.TypeRack, Key: "65537"}
		toProj   = parentOf(rootID, project)
		toRanges = parentOf(rootID, groupID(ranges))
		inRange  = parentOf(groupID(ranges), groupID(inRanges))
		groups   map[uuid.UUID]v1.Group
		logs     *observer.ObservedLogs
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		names := map[uuid.UUID]string{
			channels: "Channels", users: "Users", a: "A", b: "B", inUsers: "In Users",
			inProj: "In Project", inRack: "In Rack", nested: "Nested", orphan: "Orphan",
			ranges: "Ranges", inRanges: "In Ranges", implied: "Implied",
			inImpl: "In Implied", mixed: "Mixed", unknown: "Unknown",
			inUnkn: "In Unknown", inWs: "In Workspace", wsRoot: "Workspaces",
			inWsRoot: "In Workspaces", ofWs: "Of Workspaces", tasks: "Tasks",
			inTasks: "Task Folder",
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
			parentOf(groupID(implied), groupID(inImpl)),
			parentOf(rootID, groupID(unknown)),
			parentOf(groupID(unknown), groupID(inUnkn)),
			parentOf(resource.ID{Type: "workspace", Key: ws}, groupID(inWs)),
			parentOf(rootID, groupID(wsRoot)),
			parentOf(groupID(wsRoot), groupID(inWsRoot)),
			parentOf(rootID, groupID(tasks)),
			parentOf(groupID(tasks), groupID(inTasks)),
			toProj,
			toRanges,
			inRange,
		)
		createEdges(ctx, db, members...)
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "group",
			Migrations:      []gorp.Migration{v1.Migration, v1.ParentMigration},
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
		Entry("a group with no parent edge and no members", &orphan,
			resource.TypeBuiltin, func() *resource.ID { return nil }),
		Entry("a group with no parent edge whose members imply a scope", &implied,
			resource.TypeChannel, func() *resource.ID { return nil }),
		Entry("a group under a group placed by its members", &inImpl,
			resource.TypeChannel, func() *resource.ID { return new(groupID(implied)) }),
		Entry("a group whose members imply two scopes", &mixed, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
		Entry("a root with an unknown name", &unknown, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
		Entry("a group under an unknown root", &inUnkn, resource.TypeProject,
			func() *resource.ID { return nil }),
		Entry("a group under the legacy range root", &inRanges, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
		Entry("a group under a workspace", &inWs, resource.TypeProject,
			func() *resource.ID {
				return &resource.ID{Type: resource.TypeProject, Key: ws}
			}),
		Entry("a group under the legacy workspace root", &inWsRoot,
			resource.TypeProject, func() *resource.ID { return nil }),
		Entry("a group whose members are workspaces", &ofWs, resource.TypeProject,
			func() *resource.ID { return nil }),
		Entry("a group of tasks", &inTasks, resource.TypeBuiltin,
			func() *resource.ID { return nil }),
	)

	It("Should log each membership that the member migrations drop", func() {
		dropped := func(g uuid.UUID, name string, member resource.ID) map[string]any {
			return map[string]any{
				"group":      g.String(),
				"group_name": name,
				"member":     member.String(),
			}
		}
		logged := logs.FilterMessage(
			"dropped a member from a group that cannot hold it",
		).All()
		got := make([]map[string]any, len(logged))
		for i, e := range logged {
			got[i] = e.ContextMap()
		}
		Expect(got).To(ConsistOf(
			dropped(mixed, "Mixed", channel),
			dropped(mixed, "Mixed", schem),
			dropped(inProj, "In Project", channel),
			dropped(b, "B", twoGrps),
			dropped(inTasks, "Task Folder", task),
		))
	})

	It(
		"Should keep a member of several groups only in the first group by name",
		func(ctx SpecContext) {
			Expect(edges(ctx, db)).To(ContainElement(parentOf(groupID(a), twoGrps)))
			Expect(edges(ctx, db)).ToNot(ContainElement(parentOf(groupID(b), twoGrps)))
		},
	)

	It(
		"Should keep a pinned group out of the pick by name",
		func(ctx SpecContext) {
			Expect(edges(ctx, db)).To(ContainElements(
				parentOf(groupID(nested), subProj),
				parentOf(groupID(ofWs), subProj),
			))
		},
	)

	It("Should stage each removal with the name of its group", func(ctx SpecContext) {
		var removals []v1.Removal
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			removals, err = v1.RetrieveRemovals(ctx, tx)
			return err
		})).To(Succeed())
		Expect(removals).To(ConsistOf(
			v1.Removal{
				Group:     mixed,
				GroupName: "Mixed",
				Member:    channel,
				Reason:    v1.RemovedWithGroup,
			},
			v1.Removal{
				Group:     mixed,
				GroupName: "Mixed",
				Member:    schem,
				Reason:    v1.RemovedWithGroup,
			},
			v1.Removal{
				Group:     inProj,
				GroupName: "In Project",
				Member:    channel,
				Reason:    v1.RemovedUnheld,
			},
			v1.Removal{
				Group:     b,
				GroupName: "B",
				Member:    twoGrps,
				Reason:    v1.RemovedDuplicate,
			},
			v1.Removal{
				Group:     inTasks,
				GroupName: "Task Folder",
				Member:    task,
				Reason:    v1.RemovedWithGroup,
			},
		))
	})

	It(
		"Should delete the parent edges into groups and the overruled memberships",
		func(ctx SpecContext) {
			var parentEdges []ontology.Relationship
			for _, e := range edges(ctx, db) {
				if e.Type == ontology.RelationshipTypeParentOf {
					parentEdges = append(parentEdges, e)
				}
			}
			kept := slices.DeleteFunc(
				slices.Clone(members),
				func(e ontology.Relationship) bool {
					return e == parentOf(groupID(b), twoGrps)
				},
			)
			Expect(parentEdges).To(ConsistOf(
				append([]ontology.Relationship{toProj, toRanges, inRange}, kept...),
			))
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
		chans    = v1.Group{Key: uuid.New(), Name: "Chans", Scope: resource.TypeChannel}
		logs     = []log.Log{
			{Key: uuid.New(), Name: "In root"},
			{Key: uuid.New(), Name: "In top"},
			{Key: uuid.New(), Name: "In project"},
			{Key: uuid.New(), Name: "Owned"},
			{Key: uuid.New(), Name: "In channel group"},
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
		StartRun(db.Run)
		projTop.Parent = &project
		projLeaf.Parent = new(groupID(projTop.Key))
		groups := []v1.Group{root, top, projTop, projLeaf, chans}
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
			parentOf(groupID(chans.Key), logID(4)),
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

	It(
		"Should leave a member under a group of another scope with no group",
		func(ctx SpecContext) {
			Expect(migrate(ctx, false)[4].Group).To(BeNil())
		},
	)

	It(
		"Should move a rack from a group under a rack to its other group",
		func(ctx SpecContext) {
			rack := resource.ID{Type: resource.TypeRack, Key: "65537"}
			bench := v1.Group{
				Key:    uuid.New(),
				Name:   "Bench",
				Scope:  resource.TypeDevice,
				Parent: &rack,
			}
			loose := v1.Group{
				Key:   uuid.New(),
				Name:  "Loose",
				Scope: resource.TypeDevice,
			}
			Expect(gorp.NewCreate[v1.Key, v1.Group]().
				Entries(&[]v1.Group{bench, loose}).
				Exec(ctx, db)).To(Succeed())
			member := resource.ID{Type: resource.TypeRack, Key: logs[0].Key.String()}
			createEdges(ctx, db,
				parentOf(groupID(bench.Key), member),
				parentOf(groupID(loose.Key), member),
			)
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "rack",
				Migrations: []gorp.Migration{
					v1.NewMembersMigration(v1.MembersConfig[log.Key, log.Log]{
						Name:     "members",
						Type:     resource.TypeRack,
						SetGroup: func(l *log.Log, g v1.Key) { l.Group = &g },
					}),
				},
			})).To(Succeed())
			var l log.Log
			Expect(gorp.NewRetrieve[log.Key, log.Log]().
				Where(gorp.MatchKeys[log.Key, log.Log](logs[0].Key)).
				Entry(&l).
				Exec(ctx, db)).To(Succeed())
			Expect(l.Group).To(HaveValue(Equal(loose.Key)))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				Expect(v1.RetrieveRemovals(ctx, tx)).To(ConsistOf(v1.Removal{
					Group:     bench.Key,
					GroupName: "Bench",
					Member:    member,
					Reason:    v1.RemovedPinned,
				}))
				return nil
			})).To(Succeed())
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

var _ = Describe("TopParent", func() {
	var (
		r     = resource.ID{Type: resource.TypeRack, Key: "65537"}
		outer = v1.Group{Key: uuid.New(), Parent: &r}
		inner = v1.Group{Key: uuid.New(), Parent: new(groupID(outer.Key))}
		byKey = map[string]v1.Group{
			outer.Key.String(): outer,
			inner.Key.String(): inner,
		}
	)

	It("Should return the parent at the top of the chain", func() {
		top, ok := v1.TopParent(inner, byKey)
		Expect(ok).To(BeTrue())
		Expect(top).To(Equal(r))
	})

	It("Should return false for a group with no parent", func() {
		_, ok := v1.TopParent(v1.Group{Key: uuid.New()}, byKey)
		Expect(ok).To(BeFalse())
	})

	It("Should return false when a parent group does not exist", func() {
		g := v1.Group{Key: uuid.New(), Parent: new(groupID(uuid.New()))}
		_, ok := v1.TopParent(g, byKey)
		Expect(ok).To(BeFalse())
	})

	It("Should return false for a cycle", func() {
		a, b := uuid.New(), uuid.New()
		cycle := map[string]v1.Group{
			a.String(): {Key: a, Parent: new(groupID(b))},
			b.String(): {Key: b, Parent: new(groupID(a))},
		}
		_, ok := v1.TopParent(cycle[a.String()], cycle)
		Expect(ok).To(BeFalse())
	})
})
