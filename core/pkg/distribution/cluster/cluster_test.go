// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Host", func() {
	host := cluster.Host{Node: node.Node{Key: 1, Address: "localhost:9090"}}

	It("Should return the host node and its key", func() {
		Expect(host.Host()).To(Equal(host.Node))
		Expect(host.HostKey()).To(Equal(node.Key(1)))
	})

	It("Should resolve the address of the host node", func() {
		Expect(host.Resolve(1)).To(Equal(address.Address("localhost:9090")))
	})

	It("Should return ErrNodeNotFound for any other node", func() {
		Expect(host.Resolve(2)).Error().To(MatchError(cluster.ErrNodeNotFound))
	})
})
