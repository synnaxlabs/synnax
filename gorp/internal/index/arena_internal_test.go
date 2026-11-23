// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

// The arena and keys of a delta have no exported reader, so this spec is internal.

import (
	"strconv"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type arenaScope struct{ cleanups []func(committed bool) }

func (s *arenaScope) OnCleanup(fn func(committed bool)) {
	s.cleanups = append(s.cleanups, fn)
}

func (s *arenaScope) Committing() bool { return false }

func (s *arenaScope) end(committed bool) {
	for _, fn := range s.cleanups {
		fn(committed)
	}
	s.cleanups = nil
}

var _ = ginkgo.Describe("Delta arena across transactions", func() {
	ginkgo.It("Should keep the arena and keys of a reused delta bounded", func() {
		l := NewLookup[*arenaScope, int, string]()
		_, finish := l.Populate()
		finish(nil)
		for tx := range 2000 {
			s := &arenaScope{}
			for k := range 100 {
				l.Stage(s, k, strconv.Itoa(tx*100+k))
			}
			s.end(tx%2 == 0)
		}
		d := l.overlay.d
		gomega.Expect(d.arena).To(gomega.BeEmpty())
		gomega.Expect(d.keys).To(gomega.BeEmpty())
		gomega.Expect(cap(d.arena)).To(gomega.BeNumerically("<=", retained))
		gomega.Expect(cap(d.keys)).To(gomega.BeNumerically("<=", retained))
	})
})
