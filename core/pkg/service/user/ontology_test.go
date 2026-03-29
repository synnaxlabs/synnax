// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Ontology", Ordered, func() {
	var (
		db      *gorp.DB
		svc     *user.Service
		userKey uuid.UUID
		otg     *ontology.Ontology
	)
	BeforeAll(func(ctx SpecContext) {
		userKey = uuid.New()
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustSucceed(search.Open())
		DeferCleanup(func() {
			Expect(searchIdx.Close()).To(Succeed())
		})
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		svc = MustSucceed(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
	})
	AfterAll(func(ctx SpecContext) {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Schema", func() {
		It("Should return the ontology schema", func(ctx SpecContext) {
			schema := svc.Schema().Shape()
			Expect(schema.DataType()).To(Equal(zyn.ObjectT))
			fields := schema.Fields()
			Expect(fields).To(HaveKey("key"))
			Expect(fields).To(HaveKey("username"))
		})
	})
	Describe("retrieveResource", func() {
		It("Should retrieve a users schema entity by its key", func(ctx SpecContext) {
			u := user.User{Username: "test", Key: userKey}
			w := svc.NewWriter(nil)
			Expect(w.Create(ctx, &u)).To(Succeed())
			resource := MustSucceed(svc.RetrieveResource(ctx, userKey.String(), nil))
			var resU user.User
			Expect(resource.Parse(&resU)).To(Succeed())
			Expect(resU).To(Equal(u))
		})
	})

})
