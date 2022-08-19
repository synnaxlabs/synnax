package gorp_test

import (
	"github.com/arya-analytics/x/binary"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Entries", func() {
	Describe("Get and Set", func() {
		It("Should return an empty slice if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int, entry]()
			entries := gorp.GetEntries[int, entry](q)
			Expect(entries.All()).To(HaveLen(0))
		})
		It("Should panick if a caller attempts to set multiple entries on a single entry query", func() {
			q := query.New()
			gorp.SetEntry[int, entry](q, &entry{})
			e := gorp.GetEntries[int, entry](q)
			Expect(func() {
				e.Set(2, entry{})
			}).To(Panic())
		})
	})
	Describe("TypePrefix", func() {
		It("Should not append a type prefix to a particular key when type prefix is off", func() {
			db := memkv.New()
			gorpDB := gorp.Wrap(db,
				gorp.WithoutTypePrefix(),
			)
			Expect(gorp.NewCreate[int, entry]().
				Entries(&[]entry{{ID: 1, Data: "data"}}).
				Exec(gorpDB)).To(Succeed())
			// use msgpack to encode the entry int 1  into a byte slice
			ecd := &binary.MsgPackEncoderDecoder{}
			b, err := ecd.Encode(1)
			Expect(err).To(Not(HaveOccurred()))
			_, err = db.Get(b)
			Expect(err).To(Not(HaveOccurred()))
		})
	})

})
