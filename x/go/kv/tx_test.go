package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tx", Ordered, func() {
	var db kv.DB
	BeforeAll(func() { db = memkv.New() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("WithTx", func() {
		It("Should commit the transaction if the returned error is nil", func() {
			k := []byte("test-1")
			v := []byte("value")
			Expect(kv.WithTx(ctx, db, func(tx kv.Tx) error {
				return tx.Set(ctx, k, v)
			})).To(Succeed())
			ov, closer := MustSucceed2(db.Get(ctx, k))
			Expect(ov).To(Equal([]byte("value")))
			Expect(closer.Close()).To(Succeed())
		})
		It("Should rollback the transaction if the returned error is not nil", func() {
			k := []byte("test-2")
			err := errors.New("test error")
			Expect(kv.WithTx(ctx, db, func(tx kv.Tx) error {
				Expect(tx.Set(ctx, k, []byte("value"))).To(Succeed())
				return err
			})).To(MatchError(err))
			_, _, err = db.Get(ctx, k)
			Expect(err).To(MatchError(kv.NotFound))
		})
	})
})
