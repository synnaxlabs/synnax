// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sync_test

import (
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xsync "github.com/synnaxlabs/x/sync"
)

var _ = Describe("KeyedMutex", func() {
	var mtx xsync.KeyedMutex[string]

	BeforeEach(func() { mtx = xsync.KeyedMutex[string]{} })

	It("Should run callbacks for one key one at a time", func() {
		var (
			wg         sync.WaitGroup
			active     atomic.Int32
			overlapped atomic.Bool
		)
		for range 16 {
			wg.Go(func() {
				defer GinkgoRecover()
				Expect(mtx.Do("a", func() error {
					if active.Add(1) > 1 {
						overlapped.Store(true)
					}
					time.Sleep(time.Millisecond)
					active.Add(-1)
					return nil
				})).To(Succeed())
			})
		}
		wg.Wait()
		Expect(overlapped.Load()).To(BeFalse())
	})

	It("Should run callbacks for different keys concurrently", func() {
		var (
			wg      sync.WaitGroup
			held    = make(chan struct{})
			release = make(chan struct{})
			done    = make(chan struct{})
		)
		wg.Go(func() {
			defer GinkgoRecover()
			Expect(mtx.Do("a", func() error {
				close(held)
				<-release
				return nil
			})).To(Succeed())
		})
		Eventually(held).Should(BeClosed())
		wg.Go(func() {
			defer GinkgoRecover()
			Expect(mtx.Do("b", func() error { return nil })).To(Succeed())
			close(done)
		})
		Eventually(done).Should(BeClosed())
		close(release)
		wg.Wait()
	})

	It("Should propagate the callback error", func() {
		fnErr := errors.New("fn failed")
		Expect(mtx.Do("a", func() error { return fnErr })).To(MatchError(fnErr))
	})

	It("Should serialize again after an error released the key", func() {
		Expect(mtx.Do("a", func() error { return errors.New("boom") })).
			To(MatchError(ContainSubstring("boom")))
		Expect(mtx.Do("a", func() error { return nil })).To(Succeed())
	})
})
