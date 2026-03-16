// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("Counter", Ordered, func() {
	var db kv.DB
	BeforeAll(func() {
		db = memkv.New()
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("AtomicInt64Counter", func() {
		Context("Name Counter", Ordered, func() {
			var c *kv.AtomicInt64Counter
			BeforeAll(func() {
				var err error
				c, err = kv.OpenCounter(ctx, db, []byte("test"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a counter with a starting value of 0", func() {
				Expect(c.Value()).To(Equal(int64(0)))
			})
			It("Should increment the counter correctly", func() {
				Expect(c.Add(1)).To(Equal(int64(1)))
			})
			It("Should increment the number by a set value", func() {
				Expect(c.Add(10)).To(Equal(int64(11)))
			})
			It("Should set the counter value directly", func() {
				Expect(c.Set(421)).To(Succeed())
				Expect(c.Value()).To(Equal(int64(421)))
			})
		})
		Context("Existing Counter", func() {
			It("Should load the value of the existing counter", func() {
				c, err := kv.OpenCounter(ctx, db, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Value()).To(Equal(int64(0)))
				_, err = c.Add(10)
				Expect(err).NotTo(HaveOccurred())
				_, err = c.Add(10)
				Expect(err).NotTo(HaveOccurred())
				cTwo, err := kv.OpenCounter(ctx, db, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(cTwo.Value()).To(Equal(int64(20)))
			})
		})
	})
})
