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
			entries := gorp.GetEntries[int32, entry](q.Params)
			Expect(entries.All()).To(BeEmpty())
		})
	})

	Describe("Get + Set", func() {
		Context("Single Entry", func() {
			It("Should return the entry that was set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				gorp.SetEntry(q.Params, &entry{ID: 1})
				entries := gorp.GetEntries[int32, entry](q.Params)
				entries.Set(0, entry{ID: 2})
				Expect(entries.All()).To(HaveLen(1))
			})
		})
		Context("Multiple Entries", func() {
			It("Should return the entry that was set", func() {
				q := gorp.NewRetrieve[int32, entry]()
				gorp.SetEntries(q.Params, &[]entry{{ID: 1}})
				e := gorp.GetEntries[int32, entry](q.Params)
				e.Set(0, entry{ID: 2})
				Expect(e.All()).To(HaveLen(1))
				Expect(e.All()[0].ID).To(Equal(int32(2)))
			})
		})
	})

	Describe("Any", func() {
		It("Should return false if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int32, entry]()
			entries := gorp.GetEntries[int32, entry](q.Params)
			Expect(entries.Any()).To(BeFalse())
		})
		It("Should return true if entries were set on the query", func() {
			q := gorp.NewRetrieve[int32, entry]()
			gorp.SetEntry(q.Params, &entry{ID: 1})
			entries := gorp.GetEntries[int32, entry](q.Params)
			Expect(entries.Any()).To(BeTrue())
		})
	})

	Describe("MapInPlace", func() {
		Context("Multiple entries", func() {
			It("Should map the entries in place", func() {
				q := gorp.NewRetrieve[int32, entry]()
				entries := gorp.GetEntries[int32, entry](q.Params)
				entries.Add(entry{ID: 1})
				entries.Add(entry{ID: 2})
				Expect(entries.MapInPlace(func(e entry) (entry, bool, error) {
					return e, e.ID == 2, nil
				})).To(Succeed())
				Expect(entries.All()).To(HaveLen(1))
				Expect(entries.All()[0].ID).To(Equal(int32(2)))
			})
		})
		Context("Single entry", func() {
			It("Should map the entry in place", func() {
				q := gorp.NewRetrieve[int32, entry]()
				gorp.SetEntry(q.Params, &entry{ID: 2})
				entries := gorp.GetEntries[int32, entry](q.Params)
				Expect(entries.MapInPlace(func(e entry) (entry, bool, error) {
					return e, e.ID == 2, nil
				})).To(Succeed())
				Expect(entries.All()).To(HaveLen(1))
			})
			It("Should remove the entry if the map function returns false", func() {
				q := gorp.NewRetrieve[int32, entry]()
				gorp.SetEntry(q.Params, &entry{ID: 2})
				entries := gorp.GetEntries[int32, entry](q.Params)
				Expect(entries.MapInPlace(func(e entry) (entry, bool, error) {
					return e, false, nil
				})).To(Succeed())
				Expect(entries.Any()).To(BeFalse())
			})
		})
	})

	Describe("Keys", func() {
		It("Should return the keys of the entries", func() {
			q := gorp.NewRetrieve[int32, entry]()
			gorp.SetEntry(q.Params, &entry{ID: 1})
			entries := gorp.GetEntries[int32, entry](q.Params)
			Expect(entries.Keys()).To(ConsistOf(1))
		})
	})
})
