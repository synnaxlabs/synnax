// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package proxy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
)

const (
	host  node.Key = 1
	peerA node.Key = 2
	peerB node.Key = 3
)

// entry is a minimal proxy.Entry whose lease can be set per-test. id distinguishes
// entries so order within a bucket can be asserted.
type entry struct {
	id    int
	lease node.Key
}

func (e entry) Lease() node.Key { return e.lease }

var _ = Describe("Batch", func() {
	var factory proxy.BatchFactory[entry]
	BeforeEach(func() {
		factory = proxy.BatchFactory[entry](host)
	})

	It("Should route entries leased to the host into the gateway bucket", func() {
		b := factory.Batch([]entry{{id: 1, lease: host}, {id: 2, lease: host}})
		Expect(b.Gateway).To(Equal([]entry{{id: 1, lease: host}, {id: 2, lease: host}}))
		Expect(b.Peers).To(BeEmpty())
		Expect(b.Free).To(BeEmpty())
	})

	It("Should group entries leased to peer nodes by leaseholder key", func() {
		b := factory.Batch([]entry{
			{id: 1, lease: peerA},
			{id: 2, lease: peerB},
			{id: 3, lease: peerA},
		})
		Expect(b.Peers).To(HaveLen(2))
		Expect(b.Peers[peerA]).
			To(Equal([]entry{{id: 1, lease: peerA}, {id: 3, lease: peerA}}))
		Expect(b.Peers[peerB]).To(Equal([]entry{{id: 2, lease: peerB}}))
		Expect(b.Gateway).To(BeEmpty())
		Expect(b.Free).To(BeEmpty())
	})

	It("Should route entries with a free lease into the free bucket", func() {
		b := factory.Batch([]entry{
			{id: 1, lease: node.KeyFree},
			{id: 2, lease: node.KeyFree},
		})
		Expect(b.Free).To(
			Equal([]entry{{id: 1, lease: node.KeyFree}, {id: 2, lease: node.KeyFree}}),
		)
		Expect(b.Gateway).To(BeEmpty())
		Expect(b.Peers).To(BeEmpty())
	})

	It("Should split a mixed set of entries across all three buckets", func() {
		b := factory.Batch([]entry{
			{id: 1, lease: host},
			{id: 2, lease: peerA},
			{id: 3, lease: node.KeyFree},
			{id: 4, lease: peerA},
		})
		Expect(b.Gateway).To(Equal([]entry{{id: 1, lease: host}}))
		Expect(b.Peers[peerA]).
			To(Equal([]entry{{id: 2, lease: peerA}, {id: 4, lease: peerA}}))
		Expect(b.Free).To(Equal([]entry{{id: 3, lease: node.KeyFree}}))
	})

	It("Should return empty buckets and an initialized peers map for no entries", func() {
		b := factory.Batch(nil)
		Expect(b.Gateway).To(BeEmpty())
		Expect(b.Free).To(BeEmpty())
		Expect(b.Peers).ToNot(BeNil())
		Expect(b.Peers).To(BeEmpty())
	})
})
