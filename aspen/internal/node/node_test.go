// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/version"
)

var _ = Describe("Node", func() {

	Describe("Parse", func() {
		It("Should parse the Name of a node from a string", func() {
			id, err := node.Key(0).Parse("1")
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(node.Key(1)))
		})
	})

	Describe("Group", func() {

		Describe("Filter", func() {

			It("Should filter nodes correctly", func() {
				g := node.Group{
					1: node.Node{
						Key:   1,
						State: node.StateHealthy,
					},
					2: node.Node{
						Key:   2,
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
						Key:     1,
						State:   node.StateHealthy,
						Address: "localhost:0",
					},
					2: node.Node{
						Key:     2,
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
						Key:   1,
						State: node.StateHealthy,
						Heartbeat: version.Heartbeat{
							Version:    1,
							Generation: 1,
						},
					},
					2: node.Node{
						Key:     2,
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

	Describe("CopyState", func() {

		It("Should copy a group of nodes", func() {
			g := node.Group{
				1: node.Node{
					Key:   1,
					State: node.StateHealthy,
					Heartbeat: version.Heartbeat{
						Version:    1,
						Generation: 1,
					},
				},
				2: node.Node{
					Key:     2,
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
