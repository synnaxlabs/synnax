package user_test

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	Describe("RetrieveEntity", func() {
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
