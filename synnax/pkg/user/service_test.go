// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("User", Ordered, func() {
	var (
		db      *gorp.DB
		svc     *user.Service
		otg     *ontology.Ontology
		userKey uuid.UUID
	)
	BeforeAll(func() {
		userKey = uuid.New()
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		svc = MustSucceed(user.NewService(ctx, user.Config{DB: db, Ontology: otg, Group: g}))
	})
	AfterAll(func() {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new user", func() {
			w := svc.NewWriter(nil)
			u := &user.User{Username: "test", Key: userKey}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should return an error if the user with the key already exists", func() {
			w := svc.NewWriter(nil)
			u := &user.User{Username: "test", Key: userKey}
			Expect(errors.Is(w.Create(ctx, u), query.UniqueViolation)).To(BeTrue())
		})
	})
	Describe("Update", func() {
		It("Should update the user", func() {
			w := svc.NewWriter(nil)
			u := user.User{Username: "test2"}
			Expect(w.Create(ctx, &u)).To(Succeed())
			u.Username = "test3"
			Expect(w.Update(ctx, u)).To(Succeed())
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a user by its key", func() {
			user, err := svc.Retrieve(ctx, userKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
		})
	})
	Describe("RetrieveByUsername", func() {
		It("Should retrieve a user by its username", func() {
			user, err := svc.RetrieveByUsername(ctx, "test")
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
		})
	})
	Describe("UsernameExists", func() {
		It("Should return true if the username exists", func() {
			Expect(svc.UsernameExists(ctx, "test")).To(BeTrue())
		})
		It("Should return false if the username does not exist", func() {
			Expect(svc.UsernameExists(ctx, "test2")).To(BeFalse())
		})
	})
})
