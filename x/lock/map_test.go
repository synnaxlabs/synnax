// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lock"
	"sync"
)

var _ = Describe("ApplySink", func() {
	It("Should allow the caller to acquire the lock", func() {
		m := lock.NewKeys[int]()
		Expect(m.TryLock(1)).To(BeTrue())
	})
	It("Should return an error when the caller tries to acquire a lock that is already held", func() {
		m := lock.NewKeys[int]()
		Expect(m.TryLock(1)).To(BeTrue())
		Expect(m.TryLock(1)).To(BeFalse())
	})
	It("Should allow the called to release the lock", func() {
		m := lock.NewKeys[int]()
		Expect(m.TryLock(1)).To(BeTrue())
		m.Unlock(1)
		Expect(m.TryLock(1)).To(BeTrue())
	})
	It("Should panic if the caller tries to release an unlocked lock", func() {
		m := lock.NewKeys[int]()
		Expect(func() { m.Unlock(1) }).To(Panic())
	})
	It("Should prevent multiple goroutines from acquiring the same key", func() {
		m := lock.NewKeys[int]()
		acquisitions := make([]bool, 100)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				acquisitions[i] = m.TryLock(1)
			}(i)
		}
		wg.Wait()
		totalTrue := 0
		for _, a := range acquisitions {
			if a {
				totalTrue++
			}
		}
		Expect(totalTrue).To(Equal(1))
	})
})
