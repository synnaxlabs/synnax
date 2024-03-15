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
	"github.com/synnaxlabs/x/mathutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Keys", func() {
	Describe("Key", func() {
		Describe("Construction", func() {
			It("Should return the correct leaseholder for the key", func() {
				k := channel.NewKey(core.NodeKey(1), 1)
				// print out the bytes of the key
				Expect(k.Leaseholder()).To(Equal(core.NodeKey(1)))
			})
			It("Should return the correct localKey for the key", func() {
				k := channel.NewKey(core.NodeKey(1), 2)
				Expect(k.LocalKey()).To(Equal(types.Uint20(2)))
			})
			It("Should correctly handle the maximum value of a 12 bit node key and 20 bit cesium key", func() {
				k := channel.NewKey(core.NodeKey(mathutil.MaxUint12), mathutil.MaxUint20)
				Expect(k.Leaseholder()).To(Equal(core.NodeKey(mathutil.MaxUint12)))
				Expect(k.LocalKey()).To(Equal(mathutil.MaxUint20))
			})
		})
		Describe("Lease", func() {
			It("Should return the leaseholder node Name", func() {
				k := channel.NewKey(core.NodeKey(1), 1)
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
