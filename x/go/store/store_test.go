// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package store_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/store"
)

type state struct {
	value int
}

func (s state) Copy() state {
	return s
}

func copyState(s state) state {
	return s
}

var _ = Describe("Store", func() {
	Describe("core", func() {
		It("Should initialize a basic store correctly", func() {
			s := store.New(copyState)
			state := s.CopyState()
			Expect(state.value).To(Equal(0))
		})
	})
	Describe("Observable", func() {
		It("Should initialize an observable store correctly", func() {
			s := store.ObservableWrap(store.New(copyState), store.ObservableConfig[state]{GoNotify: config.Bool(false)})
			var changedState state
			s.OnChange(func(s state) { changedState = s })
			s.SetState(state{value: 2})
			Expect(changedState.value).To(Equal(2))
		})
	})
})
