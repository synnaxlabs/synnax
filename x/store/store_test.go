package store_test

import (
	"github.com/arya-analytics/x/store"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("StorageKey", func() {
	Describe("core", func() {
		It("Should initialize a basic store correctly", func() {
			s := store.New(copyState)
			state := s.CopyState()
			Expect(state.value).To(Equal(0))
		})
	})
	Describe("Observable", func() {
		It("Should initialize an observable store correctly", func() {
			s := store.ObservableWrap(store.New(copyState))
			var changedState state
			s.OnChange(func(s state) { changedState = s })
			s.SetState(state{value: 2})
			Expect(changedState.value).To(Equal(2))
		})
	})
})
