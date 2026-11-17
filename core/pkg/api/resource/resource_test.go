// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resource", func() {
	var (
		term string
		lbl  label.Label
		rck  rack.Rack
	)
	BeforeEach(func(ctx SpecContext) {
		term = "probe" + uuid.New().String()[:8]
		lbl = label.Label{Name: term}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Label.NewWriter(tx).Create(ctx, &lbl)
		})).To(Succeed())
		rck = rack.Rack{Name: term + " rack"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Rack.NewWriter(tx).Create(ctx, &rck)
		})).To(Succeed())
	})

	Describe("Search", func() {
		It("Should return the matches of the types it searches, closest match first",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID(), rck.ResourceID())
				Expect(search(AuthedCtx(ctx, u), apiresource.SearchRequest{
					SearchTerm: term,
					Types:      []resource.Type{resource.TypeLabel, resource.TypeRack},
				})).To(Equal(apiresource.SearchResponse{Resources: []resource.Resource{
					{ID: lbl.ResourceID(), Name: lbl.Name},
					{ID: rck.ResourceID(), Name: rck.Name},
				}}))
			},
		)

		It("Should leave out the matches that the caller cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID())
				Expect(search(AuthedCtx(ctx, u), apiresource.SearchRequest{
					SearchTerm: term,
					Types:      []resource.Type{resource.TypeLabel, resource.TypeRack},
				})).To(Equal(apiresource.SearchResponse{Resources: []resource.Resource{
					{ID: lbl.ResourceID(), Name: lbl.Name},
				}}))
			},
		)

		It("Should fill a page with channels when the caller cannot read the tasks",
			func(ctx SpecContext) {
				tsk := task.Task{
					Type: pagerduty.AlertTaskType,
					Rack: svc.Rack.EmbeddedKey,
					Name: term,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Task.NewWriter(tx).Create(ctx, &tsk)
				})).To(Succeed())
				channels := make([]channel.Channel, 3)
				for i := range channels {
					channels[i] = channel.Channel{
						Name:     fmt.Sprintf("%s_%d", term, i),
						DataType: telem.Float64T,
						Virtual:  true,
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.Channel.NewWriter(tx).Create(ctx, &channels[i])
					})).
						To(Succeed())
				}
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), resource.ID{Type: resource.TypeChannel})
				res := MustSucceed(search(
					AuthedCtx(ctx, u),
					apiresource.SearchRequest{
						SearchTerm: term,
						Types: []resource.Type{
							resource.TypeChannel,
							resource.TypeTask,
						},
						Limit: 2,
					},
				))
				Expect(res.Resources).To(HaveLen(2))
				Expect(res.Resources).To(HaveEach(
					HaveField("ID.Type", resource.TypeChannel),
				))
			},
		)

		It("Should list only the resources that the caller can retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), rck.ResourceID())
				Expect(search(AuthedCtx(ctx, u), apiresource.SearchRequest{
					Types: []resource.Type{resource.TypeLabel, resource.TypeRack},
				})).To(Equal(apiresource.SearchResponse{Resources: []resource.Resource{
					{ID: rck.ResourceID(), Name: rck.Name},
				}}))
			},
		)

		It("Should refuse an unknown type", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(search(
				AuthedCtx(ctx, u),
				apiresource.SearchRequest{
					SearchTerm: term,
					Types:      []resource.Type{"turbine"},
				},
			)).Error().To(MatchError(`types: unknown resource type "turbine"`))
		})
	})

	Describe("Retrieve", func() {
		It("Should return the names of the IDs when retrieve is granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID(), rck.ResourceID())
				Expect(retrieve(
					AuthedCtx(ctx, u),
					apiresource.RetrieveRequest{
						IDs: []resource.ID{rck.ResourceID(), lbl.ResourceID()},
					},
				)).To(Equal(apiresource.RetrieveResponse{Resources: []resource.Resource{
					{ID: rck.ResourceID(), Name: rck.Name},
					{ID: lbl.ResourceID(), Name: lbl.Name},
				}}))
			},
		)

		It("Should leave out an ID that names no resource", func(ctx SpecContext) {
			u := createUser(ctx)
			missing := label.ResourceID(uuid.New())
			grant(ctx, u.ResourceID(), lbl.ResourceID(), missing)
			Expect(retrieve(
				AuthedCtx(ctx, u),
				apiresource.RetrieveRequest{IDs: []resource.ID{
					missing,
					lbl.ResourceID(),
				}},
			)).To(Equal(apiresource.RetrieveResponse{Resources: []resource.Resource{
				{ID: lbl.ResourceID(), Name: lbl.Name},
			}}))
		})

		It("Should deny the request when one resource cannot be retrieved",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID())
				Expect(retrieve(
					AuthedCtx(ctx, u),
					apiresource.RetrieveRequest{
						IDs: []resource.ID{lbl.ResourceID(), rck.ResourceID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should refuse an ID of an unknown type", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, u.ResourceID(), resource.ID{Type: resource.TypeFramer, Key: "1"})
			Expect(retrieve(
				AuthedCtx(ctx, u),
				apiresource.RetrieveRequest{
					IDs: []resource.ID{{Type: resource.TypeFramer, Key: "1"}},
				},
			)).Error().To(MatchError(`ids: unknown resource type "framer"`))
		})

		It("Should deny an ID of an unknown type that is not granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				id := resource.ID{Type: resource.TypeFramer, Key: "1"}
				Expect(retrieve(
					AuthedCtx(ctx, u),
					apiresource.RetrieveRequest{IDs: []resource.ID{id}},
				)).Error().To(MatchError(access.Denied(
					u.ResourceID(),
					access.ActionRetrieve,
					id,
				).Error()))
			},
		)
	})
})
