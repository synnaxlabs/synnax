// Copyright 2023 Synnax Labs, Inc.
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
	Describe("Get and set", func() {
		It("Should return an empty slice if no entries were set on the query", func() {
			q := gorp.NewRetrieve[int, entry]()
			entries := gorp.GetEntries[int, entry](q.Params)
			Expect(entries.All()).To(HaveLen(0))
		})
	})

	Describe("MapInPlace", func() {
		It("Should map the entries in place", func() {
			q := gorp.NewRetrieve[int, entry]()
			entries := gorp.GetEntries[int, entry](q.Params)
			entries.Add(entry{ID: 1})
			entries.Add(entry{ID: 2})
			Expect(entries.MapInPlace(func(e entry) (entry, bool, error) {
				return e, e.ID == 2, nil
			})).To(Succeed())
			Expect(entries.All()).To(HaveLen(1))
			Expect(entries.All()[0].ID).To(Equal(2))
		})
	})

})
