// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Entries", func() {
	Describe("All", func() {
		It("Should return an empty slice if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int32, entry]()
			Expect(q.GetEntries().All()).To(BeEmpty())
		})
	})

	Describe("Get + Set", func() {
		Context("Single Entry", func() {
			It("Should return the entry that was set", func() {
				q := gorp.NewRetrieve[int32, entry]().Entry(&entry{ID: 1})
				Expect(q.GetEntries().All()).To(Equal([]entry{{ID: 1}}))
				q.GetEntries().Set(0, entry{ID: 2})
				Expect(q.GetEntries().All()).To(Equal([]entry{{ID: 2}}))
			})
		})
		Context("Multiple Entries", func() {
			It("Should return the entry that was set", func() {
				q := gorp.NewRetrieve[int32, entry]().Entries(&[]entry{{ID: 1}})
				q.GetEntries().Set(0, entry{ID: 2})
				Expect(q.GetEntries().All()).To(HaveLen(1))
				Expect(q.GetEntries().All()[0].ID).To(Equal(int32(2)))
			})
		})
	})

	Describe("Bound", func() {
		It("Should return false if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int32, entry]()
			Expect(q.GetEntries().Bound()).To(BeFalse())
		})

		It("Should return true if entries were set on the query", func() {
			q := gorp.NewRetrieve[int32, entry]().Entry(&entry{ID: 1})
			Expect(q.GetEntries().Bound()).To(BeTrue())
		})

		It("Should return true when multiple entries were obund on the query", func() {
			q := gorp.NewRetrieve[int32, entry]().
				Entries(&[]entry{{ID: 1}, {ID: 2}})
			Expect(q.GetEntries().Bound()).To(BeTrue())
		})
	})

	Describe("MapInPlace", func() {
		Context("Multiple entries", func() {
			It("Should map the entries in place", func() {
				q := gorp.NewRetrieve[int32, entry]().Entries(&[]entry{{ID: 1}, {ID: 2}})
				Expect(q.GetEntries().MapInPlace(func(e entry) (entry, bool, error) {
					return e, e.ID == 2, nil
				})).To(Succeed())
				Expect(q.GetEntries().All()).To(HaveLen(1))
				Expect(q.GetEntries().All()[0].ID).To(Equal(int32(2)))
			})
		})
		Context("Single entry", func() {
			It("Should map the entry in place", func() {
				q := gorp.NewRetrieve[int32, entry]().Entry(&entry{ID: 2})
				Expect(q.GetEntries().MapInPlace(func(e entry) (entry, bool, error) {
					return e, e.ID == 2, nil
				})).To(Succeed())
				Expect(q.GetEntries().All()).To(HaveLen(1))
			})
			It("Should remove the entry if the map function returns false", func() {
				q := gorp.NewRetrieve[int32, entry]().Entry(&entry{ID: 2})
				Expect(q.GetEntries().MapInPlace(func(e entry) (entry, bool, error) {
					return e, false, nil
				})).To(Succeed())
				Expect(q.GetEntries().Bound()).To(BeFalse())
			})
		})
	})

	Describe("Keys", func() {
		It("Should return the keys of the entries", func() {
			q := gorp.NewRetrieve[int32, entry]().Entry(&entry{ID: 1})
			Expect(q.GetEntries().Keys()).To(ConsistOf(int32(1)))
		})
	})
})
