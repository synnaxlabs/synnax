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
	BeforeAll(func(ctx SpecContext) {
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
		w = svc.NewWriter(nil)
	})
	AfterAll(func(ctx SpecContext) {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new user", func(ctx SpecContext) {
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

		It("Should create a new user without a key", func(ctx SpecContext) {
			u := &user.User{Username: "test2"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test2"))
			Expect(u.FirstName).To(Equal(""))
			Expect(u.LastName).To(Equal(""))
			Expect(u.RootUser).To(BeFalse())
			users = append(users, *u)
		})

		It("Should create a user with a name", func(ctx SpecContext) {
			u := &user.User{Username: "test3", FirstName: "Patrick", LastName: "Star"}
			Expect(w.Create(ctx, u)).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal("test3"))
			Expect(u.RootUser).To(BeFalse())
			users = append(users, *u)
		})

		It("Should return an error if the user with the username already exists", func(ctx SpecContext) {
			u := &user.User{Username: "test1"}
			Expect(errors.Is(w.Create(ctx, u), auth.RepeatedUsername)).To(BeTrue())
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a user by its key", func(ctx SpecContext) {
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)).To(Succeed())
			Expect(u).To(Equal(users[0]))
		})
		It("Should retrieve multiple users by keys", func(ctx SpecContext) {
			var ret []user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key, users[1].Key).Entries(&ret).Exec(ctx, nil)).To(Succeed())
			Expect(ret).To(ConsistOf(users[0], users[1]))
		})
		It("Should return an error if the user does not exist", func(ctx SpecContext) {
			err := svc.NewRetrieve().WhereKeys(uuid.New()).Entry(nil).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})
		It("Should retrieve a user by its username", func(ctx SpecContext) {
			var u user.User
			Expect(svc.NewRetrieve().Where(user.WhereUsernames(users[0].Username)).Entry(&u).Exec(ctx, nil)).To(Succeed())
			Expect(u).To(Equal(users[0]))
		})
		It("Should retrieve multiple users by usernames", func(ctx SpecContext) {
			var ret []user.User
			Expect(svc.NewRetrieve().Where(user.WhereUsernames(users[0].Username, users[1].Username)).Entries(&ret).Exec(ctx, nil)).To(Succeed())
			Expect(ret).To(ConsistOf(users[0], users[1]))
		})
		It("Should return an error if the user does not exist", func(ctx SpecContext) {
			var u user.User
			err := svc.NewRetrieve().Where(user.WhereUsernames("test5")).Entry(&u).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("UsernameExists", func() {
		It("Should return true if the username exists", func(ctx SpecContext) {
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeTrue())
		})
		It("Should return false if the username does not exist", func(ctx SpecContext) {
			Expect(svc.UsernameExists(ctx, "This name does not exist")).To(BeFalse())
		})
	})
	Describe("ChangeUsername", func() {
		It("Should change the username of a user", func(ctx SpecContext) {
			newUsername := "newUsername"
			Expect(w.ChangeUsername(ctx, users[0].Key, newUsername)).To(Succeed())
			Expect(svc.UsernameExists(ctx, newUsername)).To(BeTrue())
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeFalse())
			users[0].Username = newUsername
		})
		It("Should return an error if the username already exists", func(ctx SpecContext) {
			Expect(errors.Is(w.ChangeUsername(ctx, users[0].Key, users[1].Username), auth.RepeatedUsername)).To(BeTrue())
		})
	})
	Describe("ChangeName", func() {
		It("Should change the names of a user", func(ctx SpecContext) {
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
		It("Should only change one name if the other is blank", func(ctx SpecContext) {
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
		It("Should delete a single user", func(ctx SpecContext) {
			Expect(w.Delete(ctx, users[0].Key)).To(Succeed())
			Expect(svc.UsernameExists(ctx, users[0].Username)).To(BeFalse())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[0].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
		})
		It("Should delete multiple users", func(ctx SpecContext) {
			Expect(w.Delete(ctx, users[1].Key, users[2].Key)).To(Succeed())
			Expect(svc.UsernameExists(ctx, users[1].Username)).To(BeFalse())
			Expect(svc.UsernameExists(ctx, users[2].Username)).To(BeFalse())
			var u user.User
			Expect(svc.NewRetrieve().WhereKeys(users[1].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
			Expect(svc.NewRetrieve().WhereKeys(users[2].Key).Entry(&u).Exec(ctx, nil)).To(HaveOccurred())
		})
		It("Should not delete the root user", func(ctx SpecContext) {
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

var _ = Describe("ProvisionRootUser", func() {
	It("Should create the root user when Auth and credentials are provided", func(ctx SpecContext) {
		testDB := gorp.Wrap(memkv.New())
		defer func() { Expect(testDB.Close()).To(Succeed()) }()
		testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: testDB}))
		defer func() { Expect(testOtg.Close()).To(Succeed()) }()
		testSearch := MustSucceed(search.Open())
		defer func() { Expect(testSearch.Close()).To(Succeed()) }()
		testGroup := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
			DB: testDB, Ontology: testOtg, Search: testSearch,
		}))
		defer func() { Expect(testGroup.Close()).To(Succeed()) }()
		authKV := MustSucceed(auth.OpenKV(ctx, auth.KVConfig{DB: testDB}))
		defer func() { Expect(authKV.Close()).To(Succeed()) }()

		svc := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
			DB:              testDB,
			Ontology:        testOtg,
			Group:           testGroup,
			Search:          testSearch,
			Auth:            authKV,
			RootCredentials: auth.InsecureCredentials{Username: "synnax", Password: "seldon"},
		}))
		defer func() { Expect(svc.Close()).To(Succeed()) }()

		var rootUser user.User
		Expect(svc.NewRetrieve().Where(user.WhereUsernames("synnax")).Entry(&rootUser).Exec(ctx, nil)).To(Succeed())
		Expect(rootUser.RootUser).To(BeTrue())
		Expect(rootUser.Key).ToNot(Equal(uuid.Nil))
	})

	It("Should be idempotent across multiple opens", func(ctx SpecContext) {
		testDB := gorp.Wrap(memkv.New())
		defer func() { Expect(testDB.Close()).To(Succeed()) }()
		testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: testDB}))
		defer func() { Expect(testOtg.Close()).To(Succeed()) }()
		testSearch := MustSucceed(search.Open())
		defer func() { Expect(testSearch.Close()).To(Succeed()) }()
		testGroup := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
			DB: testDB, Ontology: testOtg, Search: testSearch,
		}))
		defer func() { Expect(testGroup.Close()).To(Succeed()) }()
		authKV := MustSucceed(auth.OpenKV(ctx, auth.KVConfig{DB: testDB}))
		defer func() { Expect(authKV.Close()).To(Succeed()) }()

		creds := auth.InsecureCredentials{Username: "synnax", Password: "seldon"}
		svc1 := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
			DB: testDB, Ontology: testOtg, Group: testGroup, Search: testSearch,
			Auth: authKV, RootCredentials: creds,
		}))

		var rootUser user.User
		Expect(svc1.NewRetrieve().Where(user.WhereUsernames("synnax")).Entry(&rootUser).Exec(ctx, nil)).To(Succeed())
		firstKey := rootUser.Key
		Expect(svc1.Close()).To(Succeed())

		svc2 := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
			DB: testDB, Ontology: testOtg, Group: testGroup, Search: testSearch,
			Auth: authKV, RootCredentials: creds,
		}))
		defer func() { Expect(svc2.Close()).To(Succeed()) }()

		var rootUser2 user.User
		Expect(svc2.NewRetrieve().Where(user.WhereUsernames("synnax")).Entry(&rootUser2).Exec(ctx, nil)).To(Succeed())
		Expect(rootUser2.Key).To(Equal(firstKey))
	})

	It("Should not create a root user when Auth is nil", func(ctx SpecContext) {
		testDB := gorp.Wrap(memkv.New())
		defer func() { Expect(testDB.Close()).To(Succeed()) }()
		testOtg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: testDB}))
		defer func() { Expect(testOtg.Close()).To(Succeed()) }()
		testSearch := MustSucceed(search.Open())
		defer func() { Expect(testSearch.Close()).To(Succeed()) }()
		testGroup := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
			DB: testDB, Ontology: testOtg, Search: testSearch,
		}))
		defer func() { Expect(testGroup.Close()).To(Succeed()) }()

		svc := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
			DB: testDB, Ontology: testOtg, Group: testGroup, Search: testSearch,
		}))
		defer func() { Expect(svc.Close()).To(Succeed()) }()

		exists := MustSucceed(svc.UsernameExists(ctx, "synnax"))
		Expect(exists).To(BeFalse())
	})
})
