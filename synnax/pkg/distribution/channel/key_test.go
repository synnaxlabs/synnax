// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

var _ = Describe("Keys", func() {
	Describe("Keys", func() {
		Describe("Name", func() {
			It("Should create a new key with the given node Name and cesium key", func() {
				k := channel.NewKey(core.NodeKey(1), 2)
				Expect(k.Leaseholder()).To(Equal(core.NodeKey(1)))
			})
		})
		Describe("Lease", func() {
			It("Should return the leaseholder node Name", func() {
				k := channel.NewKey(core.NodeKey(1), 2)
				Expect(k.Lease()).To(Equal(k.Leaseholder()))
			})
		})
		Describe("OntologyID", func() {
			It("Should return the ontology Name for the channel", func() {
				ok := channel.OntologyID(channel.NewKey(core.NodeKey(1), 2))
				Expect(ok).To(Equal(ontology.ID{
					Type: "channel",
					Key:  channel.NewKey(core.NodeKey(1), 2).String(),
				}))
			})
		})
	})
	Describe("Keys", func() {
		Describe("UniqueNodeKeys", func() {
			It("Should return a slice of the unique node ids for a set of keys", func() {
				ids := channel.Keys{
					channel.NewKey(core.NodeKey(1), 2),
					channel.NewKey(core.NodeKey(3), 4),
					channel.NewKey(core.NodeKey(1), 2),
				}
				Expect(ids.UniqueNodeKeys()).To(Equal([]core.NodeKey{1, 3}))
			})
		})
	})
})
