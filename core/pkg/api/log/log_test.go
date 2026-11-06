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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilog "github.com/synnaxlabs/synnax/pkg/api/log"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := log.Log{Name: "unretrievable"}
		Expect(svc.Log.NewWriter(nil).Create(ctx, uuid.Nil(), &r)).To(Succeed())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apilog.RetrieveRequest{
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
		res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apilog.CreateRequest{
			Project:     proj.Key,
			Logs:        []log.Log{{Name: "broadcast"}},
			DispatchKey: "dk-create",
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.Logs[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
})

// createProjectGroup creates a project group with no top parent.
func createProjectGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeProject}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func createLog(ctx SpecContext, name string) log.Log {
	GinkgoHelper()
	l := log.Log{Name: name}
	Expect(svc.Log.NewWriter(nil).Create(ctx, proj.Key, &l)).To(Succeed())
	return l
}

func retrieveLog(ctx SpecContext, key log.Key) log.Log {
	GinkgoHelper()
	var l log.Log
	Expect(svc.Log.NewRetrieve().
		Where(log.MatchKeys(key)).
		Entry(&l).
		Exec(ctx, nil)).To(Succeed())
	return l
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			l := createLog(ctx, "Set group denied")
			g := createProjectGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), log.ResourceID(l.Key))
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
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
			g := createProjectGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				log.ResourceID(l.Key),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apilog.SetGroupRequest{Group: &g.Key, Keys: []log.Key{l.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveLog(ctx, l.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createProjectGroup(ctx, "Create denied")
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apilog.CreateRequest{
				Project: proj.Key,
				Logs:    []log.Log{{Name: "Denied", Group: &g.Key}},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the log with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createProjectGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apilog.CreateRequest{
			Project: proj.Key,
			Logs:    []log.Log{{Name: "Granted", Group: &g.Key}},
		}))
		Expect(retrieveLog(ctx, res.Logs[0].Key).Group).To(Equal(&g.Key))
	})
})
