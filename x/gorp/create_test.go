package gorp_test

import (
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type entry struct {
	ID   int
	Data string
}

func (m entry) GorpKey() int { return m.ID }

func (m entry) SetOptions() []interface{} { return nil }

var _ = Describe("Create", func() {
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
	Context("Single entry", func() {
		It("Should create the entry in the db", func() {
			e := &entry{
				ID:   42,
				Data: "The answer to life, the universe, and everything",
			}
			Expect(gorp.NewCreate[int, entry]().Entry(e).Exec(db)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(42).Exists(db)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
	Describe("Multiple entries", func() {
		It("Should create the entries in the db", func() {
			var entries []entry
			for i := 0; i < 10; i++ {
				entries = append(entries, entry{ID: i, Data: "data"})
			}
			Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(db)).To(Succeed())
			var keys []int
			for _, e := range entries {
				keys = append(keys, e.ID)
			}
			exists, err := gorp.NewRetrieve[int, entry]().
				WhereKeys(keys...).Exists(db)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
	Describe("txn", func() {
		It("Should execute operations within a transaction", func() {
			var (
				entries []entry
				keys    []int
			)
			txn := db.BeginTxn()
			for i := 0; i < 10; i++ {
				entries = append(entries, entry{ID: i, Data: "data"})
				keys = append(keys, i)
			}
			Expect(gorp.NewCreate[int, entry]().Entries(&entries).Exec(txn)).To(Succeed())
			exists, err := gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(db)
			Expect(err).To(BeNil())
			Expect(exists).To(BeFalse())
			Expect(txn.Commit()).To(Succeed())
			exists, err = gorp.NewRetrieve[int, entry]().WhereKeys(keys...).Exists(db)
			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
})
