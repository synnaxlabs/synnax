package node_test

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Node", func() {

	Describe("Parse", func() {
		It("Should parse the ID of a node from a string", func() {
			id, err := node.ID(0).Parse("1")
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(node.ID(1)))
		})
	})

	Describe("Group", func() {

		Describe("Filter", func() {

			It("Should filter nodes correctly", func() {
				g := node.Group{
					1: node.Node{
						ID:    1,
						State: node.StateHealthy,
					},
					2: node.Node{
						ID:    2,
						State: node.StateSuspect,
					},
				}
				Expect(g.WhereState(node.StateHealthy)).To(HaveLen(1))
				Expect(g.WhereActive()).To(HaveLen(2))
				Expect(g.WhereNot(1)).To(HaveLen(1))
			})

		})

		Describe("Addresses", func() {

			It("Should return the addresses of the nodes in the cluster", func() {
				g := node.Group{
					1: node.Node{
						ID:      1,
						State:   node.StateHealthy,
						Address: "localhost:0",
					},
					2: node.Node{
						ID:      2,
						State:   node.StateSuspect,
						Address: "localhost:1",
					},
				}
				for _, addr := range g.Addresses() {
					Expect(addr).To(BeElementOf([]address.Address{"localhost:0", "localhost:1"}))
				}
			})

		})

		Describe("Digests", func() {

			It("Should return the digests of the nodes", func() {
				g := node.Group{
					1: node.Node{
						ID:    1,
						State: node.StateHealthy,
						Heartbeat: version.Heartbeat{
							Version:    1,
							Generation: 1,
						},
					},
					2: node.Node{
						ID:      2,
						State:   node.StateSuspect,
						Address: "localhost:1",
						Heartbeat: version.Heartbeat{
							Version:    2,
							Generation: 2,
						},
					},
				}
				Expect(g.Digests()).To(HaveLen(2))
				Expect(g.Digests()[1]).To(Equal(g[1].Digest()))
			})

		})

	})

	Describe("Copy", func() {

		It("Should copy a group of nodes", func() {
			g := node.Group{
				1: node.Node{
					ID:    1,
					State: node.StateHealthy,
					Heartbeat: version.Heartbeat{
						Version:    1,
						Generation: 1,
					},
				},
				2: node.Node{
					ID:      2,
					State:   node.StateSuspect,
					Address: "localhost:1",
					Heartbeat: version.Heartbeat{
						Version:    2,
						Generation: 2,
					},
				},
			}
			g2 := g.Copy()
			delete(g2, 1)
			Expect(g2).To(HaveLen(1))
			Expect(g).To(HaveLen(2))
		})

	})

})
