// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type transientSegment struct {
	TransientProvider
	LinearTransform[int, int]
}

func newTransientSegment() *transientSegment {
	t := &transientSegment{}
	t.Transform = func(ctx context.Context, i int) (int, bool, error) {
		if i == 3 {
			t.Transient() <- errors.New("error")
		}
		return i * i, true, nil
	}
	return t
}

var _ = Describe("TransientProvider", func() {
	var (
		ctx    signal.Context
		cancel context.CancelFunc
		trans  Stream[error]
		inlet  Stream[int]
		outlet Stream[int]
		t      *transientSegment
	)
	BeforeEach(func() {
		t = newTransientSegment()
		trans = NewStream[error](1)
		trans.SetInletAddress("transient")
		ctx, cancel = signal.TODO()
		inlet = NewStream[int](0)
		t.InFrom(inlet)
		outlet = NewStream[int](0)
		t.OutTo(outlet)
	})

	AfterEach(func() { cancel() })

	runSpecs := func() {
		It("Should receive errors from the segment", func() {
			inlet.Inlet() <- 3
			Expect(errors.Is(<-trans.Outlet(), errors.New("error"))).To(BeTrue())
		})
		It("Should close the transient channel when the segments exit", func() {
			inlet.Close()
			_, ok := <-trans.Outlet()
			Expect(ok).To(BeFalse())
		})
	}

	Describe("Transient", func() {
		BeforeEach(func() {
			s := InjectTransient[int, int](trans, t)
			s.Flow(ctx, CloseInletsOnExit())
		})
		runSpecs()
	})

	Describe("TransientSource", func() {
		BeforeEach(func() {
			s := InjectTransientSource[int](trans, t)
			s.Flow(ctx, CloseInletsOnExit())
		})
		runSpecs()
	})

	Describe("TransientSink", func() {
		BeforeEach(func() {
			s := InjectTransientSink[int](trans, t)
			s.Flow(ctx, CloseInletsOnExit(), CancelOnExitErr())
		})
		runSpecs()
	})
})
