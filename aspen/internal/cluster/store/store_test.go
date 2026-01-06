// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/version"
)

var _ = Describe("Store", func() {

	var s store.Store

	BeforeEach(func() { s = store.New(ctx) })

	Describe("Name", func() {

		It("Should open a new Store with empty state", func() {
			Expect(s.CopyState().Nodes).ToNot(BeNil())
		})

	})

	Describe("SetNode and Node", func() {

		It("Should set a node in Store", func() {
			s.SetNode(ctx, node.Node{Key: 1})
			n, ok := s.GetNode(1)
			Expect(ok).To(BeTrue())
			Expect(n.Key).To(Equal(node.Key(1)))
		})

	})

	Describe("Apply", func() {

		It("Should add nonexistent nodes", func() {
			s.Merge(ctx, node.Group{1: node.Node{Key: 1}})
			n, ok := s.GetNode(1)
			Expect(ok).To(BeTrue())
			Expect(n.Key).To(Equal(node.Key(1)))
		})

		It("Should replaces nodes with an old heartbeat", func() {
			s.SetNode(ctx, node.Node{Key: 1})
			s.Merge(ctx, node.Group{1: node.Node{Key: 1, Heartbeat: version.Heartbeat{
				Version:    1,
				Generation: 0,
			}}})
			n, ok := s.GetNode(1)
			Expect(ok).To(BeTrue())
			Expect(n.Key).To(Equal(node.Key(1)))
			Expect(n.Heartbeat.Version).To(Equal(uint32(1)))
		})

	})

	Describe("Lease", func() {

		It("Should set and get the host correctly", func() {
			s.SetHost(ctx, node.Node{Key: 1})
			Expect(s.GetHost().Key).To(Equal(node.Key(1)))
		})

		It("Should return an empty host when not set", func() {
			Expect(s.GetHost()).To(Equal(node.Node{}))
		})

	})

})
