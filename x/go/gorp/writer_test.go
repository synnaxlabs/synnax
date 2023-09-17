package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Writer", Ordered, func() {
	var (
		db *gorp.DB
		tx gorp.Tx
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
	It("Should wrap a key-value writer with an encoder", func() {
		w := gorp.WrapWriter[int, entry](tx)
		Expect(w.Set(ctx, entry{ID: 1, Data: "Two"})).To(Succeed())
		Expect(w.Delete(ctx, 1)).To(Succeed())
	})
})
