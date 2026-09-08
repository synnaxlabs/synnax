// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions_test

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	gorptestutil "github.com/synnaxlabs/x/gorp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

type counter struct {
	Key   string `json:"key"   msgpack:"key"`
	Value int    `json:"value" msgpack:"value"`
}

func (c counter) GorpKey() string { return c.Key }

func (counter) SetOptions() []any { return nil }

var _ = Describe("Dispatch", func() {
	var (
		db  *gorp.DB
		st  *actions.State[string, testAction]
		rec *Recorder[string, testAction]
	)

	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorptestutil.OpenGorpMsgpackDB())
		st = actions.NewState[string, testAction](db)
		rec = &Recorder[string, testAction]{}
		DeferCleanup(st.OnAction(rec.Record))
		Expect(
			gorp.NewCreate[string, counter]().
				Entry(&counter{Key: "c"}).
				Exec(ctx, db),
		).To(Succeed())
	})

	increment := func(ctx SpecContext) func(gorp.Tx) error {
		return func(tx gorp.Tx) error {
			return gorp.NewUpdate[string, counter]().
				Where(gorp.MatchKeys[string, counter]("c")).
				Change(func(_ gorp.Context, c counter) counter {
					c.Value++
					return c
				}).Exec(ctx, tx)
		}
	}

	fetch := func(ctx SpecContext) counter {
		GinkgoHelper()
		var res counter
		Expect(
			gorp.NewRetrieve[string, counter]().
				Where(gorp.MatchKeys[string, counter]("c")).
				Entry(&res).
				Exec(ctx, db),
		).To(Succeed())
		return res
	}

	It("Should apply the staged change and notify it", func(ctx SpecContext) {
		acts := []testAction{{Type: "inc"}}
		Expect(st.Dispatch(ctx, "c", "dk", acts, increment(ctx))).To(Succeed())
		Expect(fetch(ctx).Value).To(Equal(1))
		seen := rec.Snapshot()
		Expect(seen).To(HaveLen(1))
		Expect(seen[0].Key).To(Equal("c"))
		Expect(seen[0].DispatchKey).To(Equal("dk"))
		Expect(seen[0].Actions).To(Equal(acts))
	})

	It(
		"Should preserve every concurrent read-modify-write dispatch",
		func(ctx SpecContext) {
			const n = 32
			var wg sync.WaitGroup
			for range n {
				wg.Go(func() {
					defer GinkgoRecover()
					Expect(st.Dispatch(
						ctx, "c", "dk", []testAction{{Type: "inc"}}, increment(ctx),
					)).To(Succeed())
				})
			}
			wg.Wait()
			Expect(fetch(ctx).Value).To(Equal(n))
			Expect(rec.Snapshot()).To(HaveLen(n))
		},
	)

	It("Should notify only after the commit", func(ctx SpecContext) {
		observed := -1
		DeferCleanup(st.OnAction(
			func(_ context.Context, _ actions.Scoped[string, testAction]) {
				observed = fetch(ctx).Value
			},
		))
		Expect(st.Dispatch(
			ctx, "c", "dk", []testAction{{Type: "inc"}}, increment(ctx),
		)).To(Succeed())
		Expect(observed).To(Equal(1))
	})

	It(
		"Should return the stage error and emit no notification",
		func(ctx SpecContext) {
			stageErr := errors.New("stage failed")
			Expect(st.Dispatch(
				ctx, "c", "dk", []testAction{{Type: "inc"}},
				func(gorp.Tx) error { return stageErr },
			)).To(MatchError(stageErr))
			Expect(fetch(ctx).Value).To(Equal(0))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
	)
})
