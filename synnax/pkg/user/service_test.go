package user_test

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StreamService", Ordered, func() {
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
	Describe("Write", func() {
		Describe("Create", func() {
			It("Should create a new user", func() {
				w := svc.NewWriter()
				u := &user.User{Username: "test", Key: userKey}
				Expect(w.Create(u)).To(Succeed())
				Expect(u.Key).ToNot(Equal(uuid.Nil))
			})
			It("Should return an error if the user with the key already exists", func() {
				w := svc.NewWriter()
				u := &user.User{Username: "test", Key: userKey}
				Expect(errors.Is(w.Create(u), query.UniqueViolation)).To(BeTrue())
			})
		})
		Describe("Update", func() {
			It("Should update the user", func() {
				w := svc.NewWriter()
				u := user.User{Username: "test2"}
				Expect(w.Create(&u)).To(Succeed())
				u.Username = "test3"
				Expect(w.Update(u)).To(Succeed())
			})
		})
	})
	Describe("Retrieve", func() {
		It("Should retrieve a user by its key", func() {
			user, err := svc.Retrieve(userKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
		})
	})
	Describe("RetrieveByUsername", func() {
		It("Should retrieve a user by its username", func() {
			user, err := svc.RetrieveByUsername("test")
			Expect(err).ToNot(HaveOccurred())
			Expect(user.Key).To(Equal(userKey))
		})
	})
	Describe("UsernameExists", func() {
		It("Should return true if the username exists", func() {
			Expect(svc.UsernameExists("test")).To(BeTrue())
		})
		It("Should return false if the username does not exist", func() {
			Expect(svc.UsernameExists("test2")).To(BeFalse())
		})
	})
})
