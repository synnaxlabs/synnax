// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

var _ = Describe("SeqNum", Ordered, func() {
	var (
		kve kv.DB
	)
	BeforeAll(func() {
		db, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
		Expect(err).NotTo(HaveOccurred())
		kve = pebblekv.Wrap(db)
	})
	AfterAll(func() {
		Expect(kve.Close()).To(Succeed())
	})
	Describe("PersistedCounter", func() {
		Context("Requests SeqNum", Ordered, func() {
			var c *kv.PersistedCounter
			BeforeAll(func() {
				var err error
				c, err = kv.OpenCounter(ctx, kve, []byte("test"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should create a counter with a starting value of 0", func() {
				Expect(c.Value()).To(Equal(int64(0)))
			})
			It("Should increment the counter correctly", func() {
				Expect(c.Add(ctx)).To(Equal(int64(1)))
			})
			It("Should increment the number by a set value", func() {
				Expect(c.Add(ctx, 10)).To(Equal(int64(11)))
			})
		})
		Context("Existing SeqNum", func() {
			It("Should load the value of the existing counter", func() {
				c, err := kv.OpenCounter(ctx, kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(c.Value()).To(Equal(int64(0)))
				_, err = c.Add(ctx, 10)
				Expect(err).NotTo(HaveOccurred())
				_, err = c.Add(ctx, 10)
				Expect(err).NotTo(HaveOccurred())
				cTwo, err := kv.OpenCounter(ctx, kve, []byte("test-two"))
				Expect(err).NotTo(HaveOccurred())
				Expect(cTwo.Value()).To(Equal(int64(20)))
			})
		})
	})
})
