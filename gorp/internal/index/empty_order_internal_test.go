// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

// The spec stops Empty between its two reads with the overlay lock, which no exported
// path reaches, so it is internal.

import (
	"runtime"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type orderScope struct{ committing bool }

func (s *orderScope) OnCleanup(func(committed bool)) {}

func (s *orderScope) Committing() bool { return s.committing }

// blockedInEmpty reports whether a goroutine waits on a lock inside Multi.Empty.
func blockedInEmpty() bool {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for g := range strings.SplitSeq(string(buf), "\n\n") {
		if strings.Contains(g, ").Empty(") && strings.Contains(g, "Semacquire") {
			return true
		}
	}
	return false
}

var _ = ginkgo.Describe("Multi Empty while a scope commits", func() {
	ginkgo.It("Should see a holder that a commit flushes between its reads", func() {
		m := NewMulti[*orderScope, int, string]()
		_, finish := m.Populate()
		finish(nil)
		s := &orderScope{}
		m.Stage(s, 1, []string{"a"})
		s.committing = true
		gomega.Expect(m.Empty(nil)).To(gomega.BeFalse())
		o := &m.overlay
		o.mu.Lock()
		empty := make(chan bool, 1)
		go func() { empty <- m.Empty(nil) }()
		gomega.Eventually(blockedInEmpty).Should(gomega.BeTrue())
		// What finish does: flush the committed changes, then empty them.
		m.flush(o.d)
		o.d.reset()
		o.scope = nil
		o.mu.Unlock()
		gomega.Expect(<-empty).To(gomega.BeFalse())
		gomega.Expect(m.Get(nil, "a")).To(gomega.ConsistOf(1))
	})
})
