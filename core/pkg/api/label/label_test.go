// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilabel "github.com/synnaxlabs/synnax/pkg/api/label"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

func createLabel(ctx SpecContext) label.Label {
	GinkgoHelper()
	l := label.Label{Name: "Nominal"}
	Expect(svc.Label.NewWriter(nil).Create(ctx, &l)).To(Succeed())
	return l
}

func retrieveLabel(ctx SpecContext, key label.Key) label.Label {
	GinkgoHelper()
	var l label.Label
	Expect(svc.Label.NewRetrieve().
		Where(label.MatchKeys(key)).
		Entry(&l).
		Exec(ctx, nil)).To(Succeed())
	return l
}

var _ = Describe("Dispatch", func() {
	rename := []label.Action{label.NewSetAction(label.SetPayload{
		Path:  []string{"name"},
		Value: "Anomaly",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), l.OntologyID())
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Anomaly"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Nominal"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), label.OntologyID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeLabel,
			})
			rec := &Recorder[label.Key, label.Action]{}
			DeferCleanup(svc.Label.ObserveActions().OnChange(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apilabel.CreateRequest{
					Labels:      []label.Label{{Name: "Created"}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Labels).To(HaveLen(1))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)
})
