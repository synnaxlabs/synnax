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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: an open whose context ends before the owner takes it", func() {
	It("Should return the error of the context, and leave no claim once Run starts",
		func() {
			s := MustSucceed(state.Open(config(&appended{}), head(5, 3)))
			cfg := state.WriterConfig{
				Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
				Clock: 5,
				Start: 10,
			}
			deadline, cancelDeadline := context.WithTimeout(
				context.Background(),
				50*time.Millisecond,
			)
			defer cancelDeadline()
			began := time.Now()
			Expect(s.OpenWriter(deadline, cfg)).Error().
				To(MatchError(context.DeadlineExceeded))
			Expect(time.Since(began)).To(BeNumerically("<", 5*time.Second))
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			Expect(s.OpenWriter(canceled, cfg)).Error().
				To(MatchError(context.Canceled))
			run(s)
			// Each open above would hold a live claim of key 5 if the owner ran it, and
			// a second live claim of a key fails at any start.
			w := MustSucceed(s.OpenWriter(context.Background(), cfg))
			Expect(w.Close()).To(Succeed())
		})

	It("Should run an open that the owner takes before its context ends", func() {
		s := MustSucceed(state.Open(config(&appended{}), head(5, 3)))
		run(s)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		w := MustSucceed(s.OpenWriter(ctx, state.WriterConfig{
			Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
			Clock: 5,
			Start: 10,
		}))
		cancel()
		Expect(s.OpenWriter(context.Background(), state.WriterConfig{
			Keys:  []state.WriterKey{{Key: 5, DataType: 4}},
			Clock: 5,
			Start: 20,
		})).Error().To(MatchError(ContainSubstring(
			"writer of key 5 starting at 20 reaches the claim of another writer " +
				"from 10",
		)))
		Expect(w.Close()).To(Succeed())
	})
})
