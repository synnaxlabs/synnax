// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

func retrieveRack(ctx SpecContext, key rack.Key) rack.Rack {
	GinkgoHelper()
	var r rack.Rack
	Expect(svc.Rack.NewRetrieve().
		Where(rack.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	return r
}

var _ = Describe("Rename", func() {
	It("Should rename the rack when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before", Integrations: []string{"ni"}}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionUpdate, u.OntologyID(), r.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveRack(ctx, r.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Integrations).To(Equal([]string{"ni"}))
	})
	It("Should keep the embedded flag of the embedded rack", func(ctx SpecContext) {
		u := createUser(ctx)
		key := svc.Rack.EmbeddedKey
		grant(ctx, access.ActionUpdate, u.OntologyID(), key.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: key, Name: "Renamed Embedded"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveRack(ctx, key)
		Expect(renamed.Name).To(Equal("Renamed Embedded"))
		Expect(renamed.Embedded).To(BeTrue())
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		grant(ctx, access.ActionUpdate, u.OntologyID(), r.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing rack", func(ctx SpecContext) {
		u := createUser(ctx)
		key := rack.NewKey(1, 60000)
		grant(ctx, access.ActionUpdate, u.OntologyID(), key.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		r := rack.Rack{Name: "Before"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apirack.RenameRequest{Key: r.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveRack(ctx, r.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each rack's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := rack.Rack{Name: "With Status"}
		withoutStatus := rack.Rack{Name: "Without Status"}
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &withStatus)).To(Succeed())
		Expect(svc.Rack.NewWriter(nil).Create(ctx, &withoutStatus)).To(Succeed())
		Expect(svc.Status.NewWriter(nil).Delete(
			ctx, rack.StatusKey(withoutStatus.Key),
		)).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			withStatus.OntologyID(), withoutStatus.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apirack.RetrieveRequest{
			Keys:          []rack.Key{withoutStatus.Key, withStatus.Key},
			IncludeStatus: true,
		}))
		Expect(res.Racks).To(HaveLen(2))
		Expect(res.Racks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Racks[0].Status).To(BeNil())
		Expect(res.Racks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Racks[1].Status.Key).To(Equal(rack.StatusKey(withStatus.Key)))
	})
})
