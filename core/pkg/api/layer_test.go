// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"context"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// newUser creates a user that may take every action on resources of types.
func newUser(ctx context.Context, types ...resource.Type) user.User {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(ctx, db, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	objects := make([]resource.ID, len(types))
	for i, t := range types {
		objects[i] = resource.ID{Type: t}
	}
	MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		svc.RBAC,
		svc.User,
		u.Key,
		access.AllActions,
		objects...,
	))
	return u
}

// execBatch decodes body as a batch and runs it as u.
func execBatch(
	ctx context.Context,
	u user.User,
	body string,
) (batch.ExecResponse, error) {
	GinkgoHelper()
	var req batch.ExecRequest
	Expect(json.Codec.Decode([]byte(body), &req)).To(Succeed())
	return exec(AuthedCtx(ctx, u), req)
}

var _ = Describe("Layer", func() {
	DescribeTable("Should refuse a batch request that writes channel storage",
		func(ctx SpecContext, endpoint string) {
			Expect(apiLayer.Batch.Exec(ctx, db, batch.ExecRequest{
				Requests: []batch.Request{{Endpoint: endpoint}},
			})).Error().To(MatchError(ContainSubstring(fmt.Sprintf(
				"requests.0.endpoint: %q cannot run in a batch",
				endpoint,
			))))
		},
		Entry("channel create", "/channel/create"),
		Entry("channel delete", "/channel/delete"),
	)

	It(
		"Should create a group and move a project into it in one transaction",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeGroup, resource.TypeProject)
			proj := project.Project{Name: "batched"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &proj)
			})).To(Succeed())
			key := uuid.New()
			MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/group/create",
					"payload": {"key": %[1]q, "name": "Batched", "scope": "project"}
				},
				{
					"endpoint": "/project/dispatch",
					"payload": {"key": %[2]q, "actions": [
						{"type": "set", "set": {"path": ["group"], "value": %[1]q}}
					]}
				}
			]}`, key, proj.Key)))
			var stored project.Project
			Expect(svc.Project.NewRetrieve().
				Where(project.MatchKeys(proj.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Group).To(Equal(&key))
		},
	)

	It(
		"Should commit nothing when the subject cannot make every change",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeGroup)
			proj := project.Project{Name: "denied"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &proj)
			})).To(Succeed())
			key := uuid.New()
			Expect(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/group/create",
					"payload": {"key": %[1]q, "name": "Denied", "scope": "project"}
				},
				{
					"endpoint": "/project/dispatch",
					"payload": {"key": %[2]q, "actions": [
						{"type": "set", "set": {"path": ["group"], "value": %[1]q}}
					]}
				}
			]}`, key, proj.Key))).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"%s cannot update %s",
					u.ResourceID(),
					proj.ResourceID(),
				)),
			))
			Expect(svc.Group.NewRetrieve().
				Where(group.MatchKeys(key)).
				Entry(&group.Group{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)

	It(
		"Should create an Arc directory and move an Arc into it in one transaction",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeArc, resource.TypeArcDirectory)
			var root directory.Directory
			Expect(svc.Arc.NewDirectoryRetrieve().
				Where(directory.MatchNested(false)).
				Entry(&root).
				Exec(ctx, db)).To(Succeed())
			a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Name:   "batched",
				Source: arc.Source{Variant: arc.GraphSource{}},
			}}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Arc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			key := uuid.New()
			MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/arc/directory/create",
					"payload": {"parent": %[1]q, "directory": {"key": %[2]q, "name": "Batched"}}
				},
				{
					"endpoint": "/arc/dispatch",
					"payload": {"key": %[3]q, "actions": [
						{"type": "set_directory", "set_directory": {"directory": %[2]q}}
					]}
				}
			]}`, root.Key, key, a.Base().Key)))
			var stored arc.Arc
			Expect(svc.Arc.NewRetrieve().
				Where(arc.MatchDirectories(key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Base().Key).To(Equal(a.Base().Key))
		},
	)

	It("Should commit nothing when a later request fails", func(ctx SpecContext) {
		u := newUser(ctx, resource.TypeGroup, resource.TypeProject)
		proj := project.Project{Name: "unbatched"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		key, missing := uuid.New(), uuid.New()
		Expect(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
			{
				"endpoint": "/group/create",
				"payload": {"key": %[1]q, "name": "Discarded", "scope": "project"}
			},
			{
				"endpoint": "/project/dispatch",
				"payload": {"key": %[2]q, "actions": [
					{"type": "set", "set": {"path": ["group"], "value": %[3]q}}
				]}
			}
		]}`, key, proj.Key, missing))).Error().To(MatchError(
			"requests.1.group: group " + missing.String() + " does not exist: " +
				"validation error",
		))
		Expect(svc.Group.NewRetrieve().
			Where(group.MatchKeys(key)).
			Entry(&group.Group{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})

	It("Should rename several channels in storage in one batch", func(ctx SpecContext) {
		u := newUser(ctx, resource.TypeChannel)
		keys := MustSucceed(CreateVirtual(ctx, db, svc.Channel, 2))
		names := []string{UniqueChannelName(), UniqueChannelName()}
		MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
			{"endpoint": "/channel/dispatch", "payload": {"key": %d, "actions": [
				{"type": "rename", "rename": {"name": %q}}
			]}},
			{"endpoint": "/channel/dispatch", "payload": {"key": %d, "actions": [
				{"type": "rename", "rename": {"name": %q}}
			]}}
		]}`, keys[0], names[0], keys[1], names[1])))
		stored := MustSucceed(node.Storage.TS.RetrieveChannels(
			ctx,
			keys[0].StorageKey(),
			keys[1].StorageKey(),
		))
		Expect(stored).To(HaveLen(2))
		Expect(stored[0].Name).To(Equal(names[0]))
		Expect(stored[1].Name).To(Equal(names[1]))
	})

	Describe("Dispatch entries", func() {
		var (
			proj                           project.Project
			sym                            symbol.Symbol
			rck                            rack.Rack
			projGroup, symGroup, rackGroup group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			proj = project.Project{Name: "moved"}
			sym = defaults.Fill(symbol.Symbol{
				Name: "moved",
				Data: symbol.Spec{
					SVG:     "<svg></svg>",
					Variant: "valve",
					States:  []symbol.State{{Key: "default", Name: "default"}},
				},
			})
			rck = rack.Rack{Name: "moved"}
			projGroup = group.Group{Name: "Projects", Scope: resource.TypeProject}
			symGroup = group.Group{Name: "Symbols", Scope: resource.TypeSchematicSymbol}
			rackGroup = group.Group{Name: "Racks", Scope: resource.TypeDevice}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				gw := svc.Group.NewWriter(tx)
				for _, g := range []*group.Group{&projGroup, &symGroup, &rackGroup} {
					if err := gw.Create(ctx, g); err != nil {
						return err
					}
				}
				if err := svc.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
					return err
				}
				if err := svc.Schematic.Symbol.NewWriter(tx).
					Create(ctx, &sym); err != nil {
					return err
				}
				return svc.Rack.NewWriter(tx).Create(ctx, &rck)
			})).To(Succeed())
		})
		// moveBody returns a batch that moves the project, symbol, and rack into their
		// groups with a set of the group on each.
		moveBody := func() string {
			return fmt.Sprintf(`{"requests": [
				{"endpoint": "/project/dispatch", "payload": {"key": %[1]q, "actions": [
					{"type": "set", "set": {"path": ["group"], "value": %[2]q}}
				]}},
				{"endpoint": "/schematic/symbol/dispatch", "payload": {
					"key": %[3]q,
					"actions": [
						{"type": "set", "set": {"path": ["group"], "value": %[4]q}}
					]
				}},
				{"endpoint": "/rack/dispatch", "payload": {"key": %[5]d, "actions": [
					{"type": "set", "set": {"path": ["group"], "value": %[6]q}}
				]}}
			]}`, proj.Key, projGroup.Key, sym.Key, symGroup.Key, rck.Key, rackGroup.Key)
		}

		It("Should move records of several types in one transaction", func(
			ctx SpecContext,
		) {
			u := newUser(
				ctx,
				resource.TypeGroup,
				resource.TypeProject,
				resource.TypeSchematicSymbol,
				resource.TypeRack,
			)
			MustSucceed(execBatch(ctx, u, moveBody()))
			var (
				storedProj project.Project
				storedSym  symbol.Symbol
				storedRack rack.Rack
			)
			Expect(svc.Project.NewRetrieve().
				Where(project.MatchKeys(proj.Key)).
				Entry(&storedProj).
				Exec(ctx, db)).To(Succeed())
			Expect(storedProj.Group).To(Equal(&projGroup.Key))
			Expect(svc.Schematic.Symbol.NewRetrieve().
				Where(symbol.MatchKeys(sym.Key)).
				Entry(&storedSym).
				Exec(ctx, db)).To(Succeed())
			Expect(storedSym.Group).To(Equal(&symGroup.Key))
			Expect(svc.Rack.NewRetrieve().
				Where(rack.MatchKeys(rck.Key)).
				Entry(&storedRack).
				Exec(ctx, db)).To(Succeed())
			Expect(storedRack.Group).To(Equal(&rackGroup.Key))
		})

		It("Should commit nothing when the subject cannot update one record", func(
			ctx SpecContext,
		) {
			u := newUser(
				ctx,
				resource.TypeGroup,
				resource.TypeProject,
				resource.TypeSchematicSymbol,
			)
			Expect(execBatch(ctx, u, moveBody())).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"%s cannot update %s",
					u.ResourceID(),
					rck.ResourceID(),
				)),
			))
			var storedProj project.Project
			Expect(svc.Project.NewRetrieve().
				Where(project.MatchKeys(proj.Key)).
				Entry(&storedProj).
				Exec(ctx, db)).To(Succeed())
			Expect(storedProj.Group).To(BeNil())
		})
	})

	It(
		"Should set a status and its labels in one transaction",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeStatus, resource.TypeLabel)
			l := label.Label{Name: "batched"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			key := uuid.New().String()
			MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/status/set",
					"payload": {"statuses": [{
						"key": %[1]q,
						"name": "Batched",
						"variant": "info",
						"message": "Batched status",
						"time": 1
					}]}
				},
				{
					"endpoint": "/label/set",
					"payload": {
						"id": {"type": "status", "key": %[1]q},
						"labels": [%[2]q],
						"replace": true
					}
				}
			]}`, key, l.Key)))
			var stored status.Status[any]
			Expect(svc.Status.NewRetrieve[any]().
				Where(status.MatchKeys[any](key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Labels).To(Equal([]label.Key{l.Key}))
		},
	)
})
