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
		w       user.Writer
	)
	BeforeAll(func() {
		userKey = uuid.New()
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(group.Config{DB: db, Ontology: otg}))
		_, err := user.NewService(ctx, user.Config{})
		Expect(err).To(HaveOccurred())
		svc = MustSucceed(user.NewService(ctx, user.Config{DB: db, Ontology: otg, Group: g}))
		w = svc.NewWriter(nil)
	})
	AfterAll(func() {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new user", func() {
			u := &user.User{Username: "test", Key: userKey}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test"))
		})
		It("Should create a new user without a key", func() {
			u := &user.User{Username: "test2"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test2"))
		})
		It("Should create a user with a name", func() {
			u := &user.User{Username: "test3", FirstName: "Patrick", LastName: "Star"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test3"))
		})
		It("Should return an error if the user with the name already exists", func() {
			u := &user.User{Username: "test"}
			Expect(errors.Is(w.Create(ctx, u), query.UniqueViolation)).To(BeTrue())
		})
	})
	Describe("Update", func() {
		It("Should update the user", func() {
			u := user.User{Username: "test4", Key: userKey, FirstName: "John", LastName: "Doe"}
			Expect(w.Update(ctx, u)).To(Succeed())
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a user by its key", func() {
			user, err := svc.Retrieve(ctx, userKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
			Expect(user.Username).To(Equal("test4"))
			Expect(user.FirstName).To(Equal("John"))
			Expect(user.LastName).To(Equal("Doe"))
		})
		It("Should return an error if the user does not exist", func() {
			_, err := svc.Retrieve(ctx, uuid.New())
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("RetrieveByUsername", func() {
		It("Should retrieve a user by its username", func() {
			user, err := svc.RetrieveByUsername(ctx, "test4")
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
			Expect(user.Username).To(Equal("test4"))
			Expect(user.FirstName).To(Equal("John"))
			Expect(user.LastName).To(Equal("Doe"))
		})
		It("Should return an error if the user does not exist", func() {
			_, err := svc.RetrieveByUsername(ctx, "test5")
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("UsernameExists", func() {
		It("Should return true if the username exists", func() {
			Expect(svc.UsernameExists(ctx, "test4")).To(BeTrue())
		})
		It("Should return false if the username does not exist", func() {
			Expect(svc.UsernameExists(ctx, "test5")).To(BeFalse())
		})
	})
})
