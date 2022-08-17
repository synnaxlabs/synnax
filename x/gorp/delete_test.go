package gorp_test

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		db   *gorp.DB
		kvDB kv.DB
	)
	BeforeEach(func() {
		kvDB = memkv.New()
		db = gorp.Wrap(kvDB)
	})
	AfterEach(func() {
		Expect(kvDB.Close()).To(Succeed())
	})
	Describe("WhereKeys", func() {
		It("Should delete an entry by key in the db", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 1, Data: "Arya"}).
				Exec(db)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().WhereKeys(1).Exec(db)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(1).Exists(db)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
		It("Should NOT return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int, entry]().WhereKeys(1).Exec(db)).To(Succeed())
		})
	})
	Describe("Where", func() {
		It("Should delete an entry by predicate in the db", func() {
			Expect(gorp.NewCreate[int, entry]().
				Entry(&entry{ID: 1, Data: "Arya"}).
				Exec(db)).To(Succeed())
			Expect(gorp.NewDelete[int, entry]().Where(func(e *entry) bool {
				return e.Data == "Arya"
			}).Exec(db)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(1).Exists(db)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
		It("Should not return an error if the entry does not exist", func() {
			Expect(gorp.NewDelete[int, entry]().Where(func(e *entry) bool {
				return e.Data == "Arya"
			}).Exec(db)).To(Succeed())
		})
	})
})
