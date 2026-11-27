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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Sealed share check", func() {
	It("Should keep a held Channel the same when a segment seals after a later one "+
		"of its key", func() {
		a := &appended{}
		h := head(5, 3)
		s := MustSucceed(state.Open(config(a), h))
		a.s, a.seq = s, h.Sequence
		run(s)
		ctx := context.Background()
		early := MustSucceed(s.OpenWriter(ctx, clockWriter(0, true, 5)))
		late := MustSucceed(s.OpenWriter(ctx, clockWriter(1000, true, 5)))
		// commitTo commits one sample at at to a new segment of w.
		commitTo := func(w *state.Writer, at int64) {
			GinkgoHelper()
			id := MustSucceed(w.NewSegment(5))
			n := MustSucceed(w.NewAlignment())
			Expect(w.Commit(commit(1, n, 0, at, at, slot(id, 8)))).To(Succeed())
		}
		// Five sealed views leave room in their array, so the seal of the open segment
		// of early into the middle of it could change it in place.
		for at := range int64(4) {
			commitTo(early, at)
			MustSucceed(early.Seal(5))
		}
		commitTo(early, 10)
		commitTo(late, 1000)
		MustSucceed(late.Seal(5))
		held := s.Channel(5)
		want := segments(held)
		Expect(want).To(HaveLen(6))
		MustSucceed(early.Seal(5))
		Expect(segments(held)).To(Equal(want))
		Expect(s.Channel(5).Len()).To(Equal(6))
		Expect(early.Close()).To(Succeed())
		Expect(late.Close()).To(Succeed())
	})
})
