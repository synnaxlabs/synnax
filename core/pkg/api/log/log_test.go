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
	"github.com/synnaxlabs/x/defaults"
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
		r := defaults.Fill(log.Log{Project: &proj.Key, Name: "unretrievable"})
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
			Logs: defaults.Fill(
				[]log.Log{{Name: "broadcast", Project: &proj.Key}},
			),
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
		l := defaults.Fill(log.Log{Key: uuid.New(), Name: "Denied", Project: &proj.Key})
		Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: defaults.Fill([]log.Log{l}),
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
			Logs: defaults.Fill(
				[]log.Log{{Key: stored.Key, Name: "Overwritten", Project: &proj.Key}},
			),
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
			Logs: defaults.Fill([]log.Log{{
				Name:     "Displayed",
				Project:  &proj.Key,
				Channels: []log.ChannelEntry{{Channel: ch.Key()}},
			}}),
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
	dispatchAs := func(ctx SpecContext, acts ...log.Action) (apilog.DispatchResponse, error) {
		return dispatch(AuthedCtx(ctx, u), apilog.DispatchRequest{
			Key:     l.Key,
			Actions: acts,
		})
	}
	rename := log.NewSetAction(
		log.SetPayload{Path: []string{"name"}, Value: "Renamed"},
	)
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
	l := defaults.Fill(log.Log{Project: &proj.Key, Name: name})
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

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createGroup(ctx, "Create denied")
			Expect(create(AuthedCtx(ctx, u), apilog.CreateRequest{
				Logs: defaults.Fill(
					[]log.Log{{Project: &proj.Key, Name: "Denied", Group: &g.Key}},
				),
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the log with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(create(AuthedCtx(ctx, u), apilog.CreateRequest{
			Logs: defaults.Fill(
				[]log.Log{{Project: &proj.Key, Name: "Granted", Group: &g.Key}},
			),
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
				Logs: defaults.Fill([]log.Log{
					{Project: carried, Name: "Elsewhere", Group: &g.Key},
				}),
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
			Logs: defaults.Fill([]log.Log{{Project: &proj.Key, Name: "Denied"}}),
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Dispatch of a move", func() {
	var (
		u      user.User
		target log.Log
		g      group.Group
		dest   project.Project
		destG  group.Group
		rec    *Recorder[log.Key, log.Action]
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		target = createLog(ctx, "Moved")
		g = createGroup(ctx, "Source")
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		destG = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Group.NewWriter(tx).Create(ctx, &destG)
		})).To(Succeed())
		rec = &Recorder[log.Key, log.Action]{}
	})
	setGroup := func(k group.Key) log.Action {
		return log.NewSetAction(log.SetPayload{Path: []string{"group"}, Value: k})
	}
	setProject := func(k project.Key) log.Action {
		return log.NewSetAction(log.SetPayload{Path: []string{"project"}, Value: k})
	}
	removeGroup := log.NewRemoveAction(log.RemovePayload{Key: "group"})
	// inGroup moves the log into g.
	inGroup := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.Log.NewWriter(tx).Apply(ctx, target.Key, setGroup(g.Key))
			return err
		})).To(Succeed())
	}
	// moveAs grants u update on the log and on objects, records the broadcasts that
	// follow, and dispatches acts as u.
	moveAs := func(
		ctx SpecContext,
		acts []log.Action,
		objects ...resource.ID,
	) (apilog.DispatchResponse, error) {
		GinkgoHelper()
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			append(objects, target.ResourceID())...,
		)
		DeferCleanup(svc.Log.OnAction(rec.Record))
		return dispatch(AuthedCtx(ctx, u), apilog.DispatchRequest{
			Key:         target.Key,
			DispatchKey: "dk-move",
			Actions:     acts,
		})
	}
	It("Should move the log into a group", func(ctx SpecContext) {
		Expect(moveAs(ctx, []log.Action{setGroup(g.Key)}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLog(ctx, target.Key).Group).To(Equal(&g.Key))
		Expect(rec.Snapshot()).To(HaveExactElements(SatisfyAll(
			HaveField("DispatchKey", "dk-move"),
			HaveField("Actions", []log.Action{setGroup(g.Key)}),
		)))
	})
	It(
		"Should move the log into the project of a group in another project",
		func(ctx SpecContext) {
			res := MustSucceed(moveAs(
				ctx,
				[]log.Action{setGroup(destG.Key)},
				destG.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]log.Action{setProject(dest.Key)}))
			moved := retrieveLog(ctx, target.Key)
			Expect(moved.Group).To(Equal(&destG.Key))
			Expect(moved.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]log.Action{setGroup(destG.Key), setProject(dest.Key)},
			)))
		},
	)
	It("Should move the log out of its group", func(ctx SpecContext) {
		inGroup(ctx)
		Expect(moveAs(ctx, []log.Action{removeGroup}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLog(ctx, target.Key).Group).To(BeNil())
	})
	It("Should move the log into another project", func(ctx SpecContext) {
		Expect(moveAs(
			ctx,
			[]log.Action{setProject(dest.Key)},
			project.ResourceID(proj.Key),
			dest.ResourceID(),
		)).To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLog(ctx, target.Key).Project).To(Equal(&dest.Key))
	})
	It(
		"Should move the log out of its group when it moves into another project",
		func(ctx SpecContext) {
			inGroup(ctx)
			res := MustSucceed(moveAs(
				ctx,
				[]log.Action{setProject(dest.Key)},
				g.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]log.Action{removeGroup}))
			moved := retrieveLog(ctx, target.Key)
			Expect(moved.Group).To(BeNil())
			Expect(moved.Project).To(Equal(&dest.Key))
		},
	)
	It("Should refuse a move into a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(moveAs(ctx, []log.Action{setGroup(missing)})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", missing)),
		))
		Expect(retrieveLog(ctx, target.Key).Group).To(BeNil())
		Expect(rec.Snapshot()).To(BeEmpty())
	})
	DescribeTable(
		"Should refuse a move without update on a record it changes",
		func(
			ctx SpecContext,
			grouped bool,
			move func() ([]log.Action, []resource.ID, resource.ID),
		) {
			if grouped {
				inGroup(ctx)
			}
			before := retrieveLog(ctx, target.Key)
			acts, granted, refused := move()
			Expect(moveAs(ctx, acts, granted...)).Error().
				To(denied(u, access.ActionUpdate, refused))
			Expect(retrieveLog(ctx, target.Key)).To(Equal(before))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"into a group without the group",
			false,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setGroup(g.Key)}, nil, g.ResourceID()
			},
		),
		Entry(
			"into a group of another project without that project",
			false,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into a group of another project without the source project",
			false,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"into another project without that project",
			false,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into another project without the source project",
			false,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setProject(dest.Key)},
					[]resource.ID{dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"out of a group without the group",
			true,
			func() ([]log.Action, []resource.ID, resource.ID) {
				return []log.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key), dest.ResourceID()},
					g.ResourceID()
			},
		),
	)
})
