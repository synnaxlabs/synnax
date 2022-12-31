// Copyright 2022 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Ontology", Ordered, func() {
	var (
		db      *gorp.DB
		svc     *user.Service
		userKey uuid.UUID
	)
	BeforeAll(func() {
		userKey = uuid.New()
		db = gorp.Wrap(memkv.New())
		otg, err := ontology.Open(db)
		Expect(err).To(BeNil())
		svc = &user.Service{DB: db, Ontology: otg}
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Schema", func() {
		It("Should return the ontology schema", func() {
			schema := svc.Schema()
			Expect(schema.Fields).To(HaveKey("key"))
			Expect(schema.Fields).To(HaveKey("username"))
		})
	})
	Describe("retrieveEntity", func() {
		It("Should retrieve a users schema entity by its key", func() {
			u := &user.User{Username: "test", Key: userKey}
			w := svc.NewWriter()
			Expect(w.Create(u)).To(Succeed())
			entity, err := svc.RetrieveEntity(userKey.String())
			Expect(err).ToNot(HaveOccurred())
			key, ok := schema.Get[uuid.UUID](entity, "key")
			Expect(ok).To(BeTrue())
			Expect(key).To(Equal(userKey))
			username, ok := schema.Get[string](entity, "username")
			Expect(ok).To(BeTrue())
			Expect(username).To(Equal("test"))
		})
	})

})
