// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createRange(ctx SpecContext) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	return r
}

func retrieveRange(ctx SpecContext, key ranger.Key) ranger.Range {
	GinkgoHelper()
	var r ranger.Range
	Expect(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	return r
}

var _ = Describe("Dispatch", func() {
	rename := []ranger.Action{ranger.NewSetAction(ranger.SetPayload{
		Path:  []string{"name"},
		Value: "Cold flow",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), r.OntologyID())
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Cold flow"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := createRange(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: r.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRange(ctx, r.Key).Name).To(Equal("Hot fire"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), ranger.OntologyID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apiranger.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeRange,
			})
			rec := &Recorder[ranger.Key, ranger.Action]{}
			DeferCleanup(svc.Ranger.OnAction(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apiranger.CreateRequest{
					Ranges: []ranger.Range{{
						Name:      "Created",
						TimeRange: telem.NewRangeSeconds(1, 5),
					}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Ranges).To(HaveLen(1))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)
})
