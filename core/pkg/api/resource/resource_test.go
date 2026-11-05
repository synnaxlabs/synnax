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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
		Expect(svc.Label.NewWriter(nil).Create(ctx, &lbl)).To(Succeed())
		rck = rack.Rack{Name: term + " rack"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &rck)).To(Succeed())
	})

	Describe("Search", func() {
		It("Should return the matches of the types it searches, closest match first",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID(), rck.ResourceID())
				Expect(apiSvc.Search(AuthedCtx(ctx, u), apiresource.SearchRequest{
					Term:  term,
					Types: []resource.Type{resource.TypeLabel, resource.TypeRack},
				})).To(Equal(apiresource.SearchResponse{Resources: []resource.Resource{
					{ID: lbl.ResourceID(), Name: lbl.Name},
					{ID: rck.ResourceID(), Name: rck.Name},
				}}))
			},
		)

		It("Should deny the request when one match cannot be retrieved",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID())
				Expect(apiSvc.Search(AuthedCtx(ctx, u), apiresource.SearchRequest{
					Term:  term,
					Types: []resource.Type{resource.TypeLabel, resource.TypeRack},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should refuse an unknown type", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(apiSvc.Search(
				AuthedCtx(ctx, u),
				apiresource.SearchRequest{
					Term:  term,
					Types: []resource.Type{"turbine"},
				},
			)).Error().To(MatchError(`types: unknown resource type "turbine"`))
		})
	})

	Describe("Retrieve", func() {
		It("Should return the names of the IDs when retrieve is granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), lbl.ResourceID(), rck.ResourceID())
				Expect(apiSvc.Retrieve(
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
			grant(ctx, u.ResourceID(), lbl.ResourceID())
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, u),
				apiresource.RetrieveRequest{IDs: []resource.ID{
					label.ResourceID(uuid.New()),
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
				Expect(apiSvc.Retrieve(
					AuthedCtx(ctx, u),
					apiresource.RetrieveRequest{
						IDs: []resource.ID{lbl.ResourceID(), rck.ResourceID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should refuse an ID of an unknown type", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(apiSvc.Retrieve(
				AuthedCtx(ctx, u),
				apiresource.RetrieveRequest{
					IDs: []resource.ID{{Type: resource.TypeFramer, Key: "1"}},
				},
			)).Error().To(MatchError(`ids: unknown resource type "framer"`))
		})
	})
})
