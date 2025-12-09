// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("User", Ordered, func() {
	var (
		db    *gorp.DB
		svc   *user.Service
		otg   *ontology.Ontology
		users []user.User
		w     user.Writer
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
		_, err := user.OpenService(ctx, user.Config{})
		Expect(err).To(HaveOccurred())
		svc = MustSucceed(user.OpenService(ctx, user.Config{DB: db, Ontology: otg, Group: g}))
		w = svc.NewWriter(nil)
	})
	AfterAll(func() {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new user", func() {
			newKey := uuid.New()
			u := &user.User{Username: "test1", Key: newKey}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).To(Equal(newKey))
			Expect(u.Username).To(Equal("test1"))
			Expect(u.FirstName).To(Equal(""))
			Expect(u.LastName).To(Equal(""))
			Expect(u.RootUser).To(BeFalse())
			users = append(users, *u)
		})
		It("Should create a new user without a key", func() {
			u := &user.User{Username: "test2"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test2"))
			Expect(u.FirstName).To(Equal(""))
			Expect(u.LastName).To(Equal(""))
			Expect(u.RootUser).To(BeFalse())
			users = append(users, *u)
		})
		It("Should create a user with a name", func() {
			u := &user.User{Username: "test3", FirstName: "Patrick", LastName: "Star"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test3"))
			Expect(u.RootUser).To(BeFalse())
			users = append(users, *u)
		})
		It("Should return an error if the user with the username already exists", func() {
			u := &user.User{Username: "test1"}
			Expect(errors.Is(w.Create(ctx, u), auth.RepeatedUsername)).To(BeTrue())
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a user by its key", func() {
			var u user.User
			err := svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(u).To(Equal(users[0]))
		})
		It("Should retrieve multiple users by keys", func() {
			var ret []user.User
			err := svc.NewRetrieve().WhereKeys(users[0].Key, users[1].Key).Entries(&ret).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ret).To(ConsistOf(users[0], users[1]))
		})
		It("Should return an error if the user does not exist", func() {
			err := svc.NewRetrieve().WhereKeys(uuid.New()).Entry(nil).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})
		It("Should retrieve a user by its username", func() {
			var user user.User
			err := svc.NewRetrieve().WhereUsernames(users[0].Username).Entry(&user).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(user).To(Equal(users[0]))
		})
		It("Should retrieve multiple users by usernames", func() {
			var ret []user.User
			err := svc.NewRetrieve().WhereUsernames(users[0].Username, users[1].Username).Entries(&ret).Exec(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(ret).To(ConsistOf(users[0], users[1]))
		})
		It("Should return an error if the user does not exist", func() {
			var user user.User
			err := svc.NewRetrieve().WhereUsernames("test5").Entry(&user).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("UsernameExists", func() {
		It("Should return true if the username exists", func() {
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeTrue())
		})
		It("Should return false if the username does not exist", func() {
			Expect(svc.UsernameExists(ctx, "This name does not exist")).To(BeFalse())
		})
	})
	Describe("ChangeUsername", func() {
		It("Should change the username of a user", func() {
			newUsername := "newUsername"
			Expect(w.ChangeUsername(ctx, users[0].Key, newUsername)).To(Succeed())
			Expect(svc.UsernameExists(ctx, newUsername)).To(BeTrue())
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeFalse())
			users[0].Username = newUsername
		})
		It("Should return an error if the username already exists", func() {
			Expect(errors.Is(w.ChangeUsername(ctx, users[0].Key, users[1].Username), auth.RepeatedUsername)).To(BeTrue())
		})
	})
	Describe("ChangeName", func() {
		It("Should change the names of a user", func() {
			newFirstName := "Patrick"
			newLastName := "Star"
			Expect(w.ChangeName(ctx, users[0].Key, newFirstName, newLastName)).To(Succeed())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)).To(Succeed())
			Expect(u.FirstName).To(Equal(newFirstName))
			Expect(u.LastName).To(Equal(newLastName))
			Expect(u.Username).To(Equal(users[0].Username))
			Expect(u.Key).To(Equal(users[0].Key))
			users[0].FirstName = newFirstName
			users[0].LastName = newLastName
		})
		It("Should only change one name if the other is blank", func() {
			newFirstName := "Spongebob"
			Expect(w.ChangeName(ctx, users[0].Key, newFirstName, "")).To(Succeed())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)).To(Succeed())
			Expect(u.FirstName).To(Equal(newFirstName))
			Expect(u.LastName).To(Equal(users[0].LastName))
			Expect(u.Username).To(Equal(users[0].Username))
			Expect(u.Key).To(Equal(users[0].Key))
			users[0].FirstName = newFirstName
		})
	})
	Describe("Delete", func() {
		It("Should delete a single user", func() {
			Expect(w.Delete(ctx, users[0].Key)).To(Succeed())
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeFalse())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
		})
		It("Should delete multiple users", func() {
			Expect(w.Delete(ctx, users[1].Key, users[2].Key)).To(Succeed())
			Expect(svc.UsernameExists(ctx, users[1].Username)).To(BeFalse())
			Expect(svc.UsernameExists(ctx, users[2].Username)).To(BeFalse())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[1].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
			Expect(svc.NewRetrieve().WhereKeys(users[2].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
		})
		It("Should not delete the root user", func() {
			u := &user.User{Username: "root", RootUser: true}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.FirstName).To(Equal(""))
			Expect(u.LastName).To(Equal(""))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("root"))
			Expect(u.RootUser).To(BeTrue())
			Expect(errors.Is(w.Delete(ctx, u.Key), errors.New("cannot delete root user"))).To(BeTrue())
		})
	})
})
