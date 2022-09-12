package store_test

import (
	"github.com/synnaxlabs/x/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
)

var _ = Describe("Store", func() {

	var s store.Store

	BeforeEach(func() { s = store.New() })

	Describe("New", func() {

		It("Should open a new Store with empty state", func() {
			Expect(s.CopyState().Nodes).ToNot(BeNil())
		})

	})

	Describe("Set and Node", func() {

		It("Should set a node in Store", func() {
			s.Set(node.Node{ID: 1})
			n, ok := s.Get(1)
			Expect(ok).To(BeTrue())
			Expect(n.ID).To(Equal(node.ID(1)))
		})

	})

	Describe("Apply", func() {

		It("Should add nonexistent nodes", func() {
			s.Merge(node.Group{1: node.Node{ID: 1}})
			n, ok := s.Get(1)
			Expect(ok).To(BeTrue())
			Expect(n.ID).To(Equal(node.ID(1)))
		})

		It("Should replaces nodes with an old heartbeat", func() {
			s.Set(node.Node{ID: 1})
			s.Merge(node.Group{1: node.Node{ID: 1, Heartbeat: version.Heartbeat{
				Version:    1,
				Generation: 0,
			}}})
			n, ok := s.Get(1)
			Expect(ok).To(BeTrue())
			Expect(n.ID).To(Equal(node.ID(1)))
			Expect(n.Heartbeat.Version).To(Equal(uint32(1)))
		})

	})

	Describe("Valid", func() {

		It("Should return false if the host is not set", func() {
			Expect(s.Valid()).To(BeFalse())
		})

		It("Should return true if the host is set", func() {
			s.SetHost(node.Node{ID: 1})
			Expect(s.Valid()).To(BeTrue())
		})

	})

	Describe("Host", func() {

		It("Should set and get the host correctly", func() {
			s.SetHost(node.Node{ID: 1})
			Expect(s.GetHost().ID).To(Equal(node.ID(1)))
		})

		It("Should return an empty host when not set", func() {
			Expect(s.GetHost()).To(Equal(node.Node{}))
		})

	})

})
