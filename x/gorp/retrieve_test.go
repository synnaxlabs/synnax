package gorp_test

import (
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/query"
	. "github.com/arya-analytics/x/testutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/arya-analytics/x/gorp"
)

var _ = Describe("RetrieveEntity", Ordered, func() {
	var (
		db      *gorp.DB
		kv      kv.DB
		entries []entry
	)
	BeforeAll(func() {
		kv = memkv.New()
		db = gorp.Wrap(kv)
		for i := 0; i < 10; i++ {
			entries = append(entries, entry{ID: i, Data: "data"})
		}
		Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(db)).To(Succeed())
	})
	AfterAll(func() { Expect(kv.Close()).To(Succeed()) })
	Describe("WhereKeys", func() {
		Context("Multiple Entries", func() {
			It("Should retrieve the entry by key", func() {
				var res []entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entries(&res).
					Exec(db)).To(Succeed())
				Expect(res).To(Equal([]entry{entries[0]}))
			})
			It("Should return a query.NotFound error if the key is not found", func() {
				var res []entry
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey(), 444444).
					Entries(&res).
					Exec(db)
				By("Returning the correct error")
				Expect(err).To(HaveOccurredAs(query.NotFound))
				By("Still retrieving as many entries as possible")
				Expect(res).To(HaveLen(1))
			})
			Describe("exists", func() {
				It("Should return true if ALL keys have matching entries", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), entries[1].GorpKey()).
						Exists(db)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if ANY key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey(), 444444).
						Exists(db)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeFalse())
				})
			})
		})
		Context("Single Entry", func() {
			It("Should retrieve the entry by key", func() {
				res := &entry{}
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(db)).To(Succeed())
				Expect(res).To(Equal(&entries[0]))
			})
			It("Should allow for a nil entry to be provided", func() {
				var res *entry
				Expect(gorp.NewRetrieve[int, entry]().
					WhereKeys(entries[0].GorpKey()).
					Entry(res).
					Exec(db)).To(Succeed())
			})
			It("Should return a query.NotFound error if the key is not found", func() {
				err := gorp.NewRetrieve[int, entry]().
					WhereKeys(444444).
					Entry(&entry{}).
					Exec(db)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.NotFound)).To(BeTrue())
			})
			Describe("exists", func() {
				It("Should return true if the key has a matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(entries[0].GorpKey()).
						Exists(db)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeTrue())
				})
				It("Should return false if the key has no matching entry", func() {
					exists, err := gorp.NewRetrieve[int, entry]().
						WhereKeys(444444).
						Exists(db)
					Expect(err).To(Not(HaveOccurred()))
					Expect(exists).To(BeFalse())
				})
			})
		})
	})
	Describe("Where", func() {
		It("Should retrieve the entry by a filter parameter", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Exec(db),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1]}))
		})
		It("Should support multiple filters", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == entries[1].ID }).
				Where(func(e *entry) bool { return e.ID == entries[2].ID }).
				Exec(db),
			).To(Succeed())
			Expect(res).To(Equal([]entry{entries[1], entries[2]}))
		})
		It("Should NOT return a query.NotFound error if no entries are found", func() {
			var res []entry
			Expect(gorp.NewRetrieve[int, entry]().
				Entries(&res).
				Where(func(e *entry) bool { return e.ID == 444444 }).
				Exec(db),
			).To(Succeed())
			Expect(res).To(HaveLen(0))
		})
		Describe("exists", func() {
			It("Should return true if ANY entries exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == entries[1].ID }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(db)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeTrue())
			})
			It("Should return false if ALL entries do not exist", func() {
				exists, err := gorp.NewRetrieve[int, entry]().
					Where(func(e *entry) bool { return e.ID == 444444 }).
					Where(func(e *entry) bool { return e.ID == 44444 }).
					Exists(db)
				Expect(err).To(Not(HaveOccurred()))
				Expect(exists).To(BeFalse())
			})
		})
	})
})
