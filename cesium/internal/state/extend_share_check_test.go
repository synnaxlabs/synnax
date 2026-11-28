// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Extend share check", func() {
	It("Should keep a held Channel the same when a later commit extends its last "+
		"pointer", func() {
		a := &appended{}
		h := head(5, 3)
		s := MustSucceed(state.Open(config(a), h))
		a.s, a.seq = s, h.Sequence
		run(s)
		w := MustSucceed(s.OpenWriter(context.Background(), clockWriter(0, false, 5)))
		id := MustSucceed(w.NewSegment(5))
		n := MustSucceed(w.NewAlignment())
		Expect(w.Commit(commit(1, n, 0, 10, 10, slot(id, 8)))).To(Succeed())
		held := s.Channel(5)
		want := slices.Clone(held.Segments[0].Pointers)
		Expect(want).To(HaveLen(1))
		// The alignment and the stream range follow the last pointer, so the commit
		// extends it.
		Expect(w.Commit(commit(1, n, 1, 11, 11, slot(id, 16)))).To(Succeed())
		Expect(held.Segments[0].Pointers).To(Equal(want))
		ptrs := s.Channel(5).Segments[0].Pointers
		Expect(ptrs).To(HaveLen(1))
		Expect(ptrs[0].Samples).To(Equal(uint32(2)))
		Expect(w.Close()).To(Succeed())
	})
})
