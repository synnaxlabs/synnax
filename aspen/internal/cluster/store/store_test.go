package store_test

import (
	"github.com/arya-analytics/aspen/internal/cluster/store"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {

	var s store.Store

	BeforeEach(func() { s = store.New() })

	Describe("New", func() {

		It("Should open a new store with empty state", func() {
			Expect(s.CopyState().Nodes).ToNot(BeNil())
		})

	})

	Describe("Set and Node", func() {

		It("Should set a node in store", func() {
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
	Describe("Flush and Load", func() {

		It("Should correctly sync the store's state to storage", func() {
			kve := memkv.New()
			s.SetHost(node.Node{ID: 1})
			s.Set(node.Node{ID: 2})
			Expect(kv.Flush(kve, []byte("key"), s)).To(Succeed())
			load := store.New()
			Expect(kv.Load(kve, []byte("key"), load)).To(Succeed())
			Expect(load.GetHost().ID).To(Equal(node.ID(1)))
			n, ok := load.Get(2)
			Expect(ok).To(BeTrue())
			Expect(n.ID).To(Equal(node.ID(2)))
		})

	})

})
