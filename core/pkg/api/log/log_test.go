// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apilog "github.com/synnaxlabs/synnax/pkg/api/log"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// createChannel creates a virtual channel for a log to display.
func createChannel(ctx SpecContext) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     "logged_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
		DataType: telem.Float64T,
		Virtual:  true,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Channel.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}

var _ = Describe("Retrieve", func() {
	It("Should leave out the logs that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		granted := createLog(ctx, "leave-out-granted")
		createLog(ctx, "leave-out-denied")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apilog.RetrieveRequest{}))
		Expect(res.Logs).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := log.Log{Project: &proj.Key, Name: "unretrievable"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Log.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		Expect(retrieve(AuthedCtx(ctx, author), apilog.RetrieveRequest{
			Where: &log.Where{Key: &filter.Set[log.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Create", func() {
	It("Should stamp the dispatch key onto the create broadcast", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		rec := &Recorder[log.Key, log.Action]{}
		DeferCleanup(svc.Log.OnAction(rec.Record))
		res := MustSucceed(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs:        []log.Log{{Name: "broadcast", Project: &proj.Key}},
			DispatchKey: "dk-create",
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.Logs[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
	It("Should refuse the create without create on the log", func(ctx SpecContext) {
		u := newUser(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), project.ResourceID(proj.Key))
		l := log.Log{Key: uuid.New(), Name: "Denied", Project: &proj.Key}
		Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: []log.Log{l},
		})).Error().To(denied(u, access.ActionCreate, l.ResourceID()))
		Expect(svc.Log.NewRetrieve().
			Where(log.MatchKeys(l.Key)).
			Entry(&log.Log{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse a log whose key exists", func(ctx SpecContext) {
		u := newCreator(ctx)
		stored := createLog(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: []log.Log{{Key: stored.Key, Name: "Overwritten", Project: &proj.Key}},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("log %s already exists", stored.Key)),
		))
		Expect(retrieveLog(ctx, stored.Key).Name).To(Equal("Stored"))
	})
	It("Should require retrieve on each channel the log displays", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		ch := createChannel(ctx)
		req := apilog.CreateRequest{
			Logs: []log.Log{{
				Name:     "Displayed",
				Project:  &proj.Key,
				Channels: []log.ChannelEntry{{Channel: ch.Key()}},
			}},
		}
		Expect(create(AuthedCtx(ctx, u), req)).Error().
			To(denied(u, access.ActionRetrieve, channel.ResourceID(ch.Key())))
		grant(ctx, access.ActionRetrieve, u.ResourceID(), channel.ResourceID(ch.Key()))
		MustSucceed(create(AuthedCtx(ctx, u), req))
	})
})

var _ = Describe("Delete", func() {
	var (
		u user.User
		l log.Log
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		l = createLog(ctx, "Deleted")
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apilog.DeleteRequest{Keys: []log.Key{l.Key}})
	}
	logID := func() resource.ID { return l.ResourceID() }
	projectID := func() resource.ID { return project.ResourceID(proj.Key) }
	DescribeTable(
		"Should refuse the delete and keep the log",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, granted, u.ResourceID(), grantedOn())
			Expect(deleteAs(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveLog(ctx, l.Key).Key).To(Equal(l.Key))
		},
		Entry(
			"without delete on the log",
			access.ActionUpdate,
			projectID,
			access.ActionDelete,
			logID,
		),
		Entry(
			"without update on its project",
			access.ActionDelete,
			logID,
			access.ActionUpdate,
			projectID,
		),
	)
	It(
		"Should delete the log with delete on it and update on its project",
		func(ctx SpecContext) {
			grant(ctx, access.ActionDelete, u.ResourceID(), logID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), projectID())
			MustSucceed(deleteAs(ctx))
			Expect(svc.Log.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&log.Log{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Dispatch", func() {
	var (
		u user.User
		l log.Log
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		l = createLog(ctx, "Dispatched")
	})
	dispatchAs := func(ctx SpecContext, acts ...log.Action) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apilog.DispatchRequest{
			Key:     l.Key,
			Actions: acts,
		})
	}
	rename := log.NewRenameAction(log.RenamePayload{Name: "Renamed"})
	It(
		"Should refuse a dispatch without update on the log and write nothing",
		func(ctx SpecContext) {
			Expect(dispatchAs(ctx, rename)).Error().
				To(denied(u, access.ActionUpdate, l.ResourceID()))
			Expect(retrieveLog(ctx, l.Key).Name).To(Equal("Dispatched"))
		},
	)
	It("Should apply the actions with update on the log", func(ctx SpecContext) {
		grant(ctx, access.ActionUpdate, u.ResourceID(), l.ResourceID())
		MustSucceed(dispatchAs(ctx, rename))
		Expect(retrieveLog(ctx, l.Key).Name).To(Equal("Renamed"))
	})
	It("Should require retrieve on a channel that an action adds", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), l.ResourceID())
		add := log.NewAddChannelAction(log.AddChannelPayload{Channel: ch.Key()})
		Expect(dispatchAs(ctx, add)).Error().
			To(denied(u, access.ActionRetrieve, channel.ResourceID(ch.Key())))
		Expect(retrieveLog(ctx, l.Key).Channels).To(BeEmpty())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), channel.ResourceID(ch.Key()))
		MustSucceed(dispatchAs(ctx, add))
		Expect(retrieveLog(ctx, l.Key).Channels).To(HaveExactElements(
			HaveField("Channel", ch.Key()),
		))
	})
})

// createGroup creates a group in the suite project.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(project.ResourceID(proj.Key))}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Group.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

func createLog(ctx SpecContext, name string) log.Log {
	GinkgoHelper()
	l := log.Log{Project: &proj.Key, Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Log.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	return l
}

func retrieveLog(ctx SpecContext, key log.Key) log.Log {
	GinkgoHelper()
	var l log.Log
	Expect(svc.Log.NewRetrieve().
		Where(log.MatchKeys(key)).
		Entry(&l).
		Exec(ctx, db)).To(Succeed())
	return l
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			l := createLog(ctx, "Set group denied")
			g := createGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), log.ResourceID(l.Key))
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apilog.SetGroupRequest{Group: &g.Key, Keys: []log.Key{l.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveLog(ctx, l.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the log with update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			l := createLog(ctx, "Set group granted")
			g := createGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				log.ResourceID(l.Key),
				g.ResourceID(),
			)
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apilog.SetGroupRequest{Group: &g.Key, Keys: []log.Key{l.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveLog(ctx, l.Key).Group).To(Equal(&g.Key))
		},
	)
	Context("Into a group under another project", func() {
		var (
			u    user.User
			x    log.Log
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			x = createLog(ctx, "Cross project")
			dest = project.Project{Name: "Destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return setGroup(
				AuthedCtx(ctx, u),
				apilog.SetGroupRequest{Group: &g.Key, Keys: []log.Key{x.Key}},
			)
		}
		DescribeTable(
			"Should refuse the move without update on both projects",
			func(ctx SpecContext, projects func() []resource.ID) {
				objects := projects()
				grant(ctx, access.ActionUpdate, u.ResourceID(), append(
					objects,
					log.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				Expect(move(ctx)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveLog(ctx, x.Key).Project).To(Equal(&proj.Key))
			},
			Entry("without the destination", func() []resource.ID {
				return []resource.ID{project.ResourceID(proj.Key)}
			}),
			Entry("without the source", func() []resource.ID {
				return []resource.ID{dest.ResourceID()}
			}),
		)
		It("Should move the log with update on both projects", func(ctx SpecContext) {
			objects := []resource.ID{project.ResourceID(proj.Key), dest.ResourceID()}
			grant(ctx, access.ActionUpdate, u.ResourceID(), append(
				objects,
				log.ResourceID(x.Key),
				g.ResourceID(),
			)...)
			MustSucceed(move(ctx))
			Expect(retrieveLog(ctx, x.Key).Project).To(Equal(&dest.Key))
		})
	})
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createGroup(ctx, "Create denied")
			Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
				Logs: []log.Log{{Project: &proj.Key, Name: "Denied", Group: &g.Key}},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the log with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: []log.Log{{Project: &proj.Key, Name: "Granted", Group: &g.Key}},
		}))
		Expect(retrieveLog(ctx, res.Logs[0].Key).Group).To(Equal(&g.Key))
	})
	Context("Under another project", func() {
		var (
			u    user.User
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newCreator(ctx)
			dest = project.Project{Name: "Create destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Create destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		createIn := func(
			ctx SpecContext,
			carried *project.Key,
		) (apilog.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apilog.CreateRequest{
				Logs: []log.Log{
					{Project: carried, Name: "Elsewhere", Group: &g.Key},
				},
			})
		}
		DescribeTable(
			"Should refuse a create in a group without update on the group's project",
			func(ctx SpecContext, carried func() *project.Key) {
				grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
				Expect(
					createIn(ctx, carried()),
				).Error().
					To(MatchError(access.ErrDenied))
			},
			Entry(
				"with the project it carries",
				func() *project.Key { return &proj.Key },
			),
			Entry("with no project", func() *project.Key { return nil }),
		)
		It(
			"Should create the log with update on the group's project",
			func(ctx SpecContext) {
				grant(
					ctx,
					access.ActionUpdate,
					u.ResourceID(),
					g.ResourceID(),
					dest.ResourceID(),
				)
				res := MustSucceed(createIn(ctx, nil))
				Expect(retrieveLog(ctx, res.Logs[0].Key).Project).To(Equal(&dest.Key))
			},
		)
	})
})

var _ = Describe("Create with an owner", func() {
	It("Should refuse the create without update on the owner", func(ctx SpecContext) {
		u := newUser(ctx)
		grant(
			ctx,
			access.ActionCreate,
			u.ResourceID(),
			resource.ID{Type: resource.TypeLog},
		)
		Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: []log.Log{{Project: &proj.Key, Name: "Denied"}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("SetProject", func() {
	createProject := func(ctx SpecContext) project.Project {
		GinkgoHelper()
		p := project.Project{Name: "destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}
	setProject := func(
		ctx SpecContext,
		u user.User,
		to project.Key,
		key log.Key,
	) (struct{}, error) {
		GinkgoHelper()
		return setProject(AuthedCtx(ctx, u), apilog.SetProjectRequest{
			Project: to,
			Keys:    []log.Key{key},
		})
	}
	It(
		"Should refuse the move without update on the destination project",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLog(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), log.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLog(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should refuse the move without update on the source project",
		func(ctx SpecContext) {
			u := newUser(ctx)
			e := createLog(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				log.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLog(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should move the log with update on both projects",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLog(ctx, "Set project granted")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				log.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			MustSucceed(setProject(ctx, u, dest.Key, e.Key))
			Expect(retrieveLog(ctx, e.Key).Project).To(Equal(&dest.Key))
		},
	)
	It(
		"Should refuse a move out of a group the subject cannot update",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLog(ctx, "Set project grouped")
			g := createGroup(ctx, "Set project grouped")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Log.NewWriter(tx).SetGroup(ctx, &g.Key, e.Key)
			})).To(Succeed())
			grant(ctx, access.ActionUpdate, u.ResourceID(), log.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, proj.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLog(ctx, e.Key).Group).To(Equal(&g.Key))
		},
	)
})
