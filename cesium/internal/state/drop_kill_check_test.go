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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: the Drop of a key that holds nothing", func() {
	It("Should record the same kills live and in a replay from a head that lacks the "+
		"key", func(ctx SpecContext) {
		a := &appended{}
		s := MustSucceed(state.Open(config(a), head(1, 1)))
		a.s = s
		run(s)
		w := MustSucceed(s.OpenWriter(ctx, state.WriterConfig{
			Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
			Clock: 5,
			Start: 10,
		}))
		id := MustSucceed(w.NewSegment(5))
		Expect(a.durable([]log.Record{create(id, 5, 5)})).To(Succeed())
		n := MustSucceed(w.NewAlignment())
		cm := commit(3, n, 0, 10, 12, slot(id, 24))
		Expect(w.Commit(cm)).To(Succeed())
		Expect(a.durable([]log.Record{cm})).To(Succeed())
		seal := MustSucceed(w.Seal(5))
		Expect(a.durable([]log.Record{&seal})).To(Succeed())
		Expect(w.Close()).To(Succeed())
		// The Tombstone takes every sample, so the segment goes and key 5 holds
		// nothing, and a head of this state lacks key 5.
		d := MustSucceed(s.BeginDelete(5, 10, 13))
		t := &log.Tombstone{Key: 5, Start: 10, End: 13}
		Expect(d.Check(t)).To(Succeed())
		Expect(a.durable([]log.Record{t})).To(Succeed())
		Expect(d.Release()).To(Succeed())
		h := MustSucceed(s.Snapshot())
		Expect(h.Channels).To(BeEmpty())
		drop := &log.Drop{Key: 5}
		Expect(a.durable([]log.Record{drop})).To(Succeed())
		live := MustSucceed(s.Snapshot())
		r := MustSucceed(state.Open(config(&appended{}), h))
		Expect(r.Replay(h.Sequence+1, []log.Record{drop})).To(Succeed())
		run(r)
		Expect(MustSucceed(r.Snapshot()).Kills).To(Equal(live.Kills))
	})
})
