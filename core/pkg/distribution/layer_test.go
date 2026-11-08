// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package distribution_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	transporttestutil "github.com/synnaxlabs/synnax/pkg/distribution/transport/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// otherNode is the identity that Aspen stores for node 2 of a two-node cluster.
const otherNode = `{"Nodes":{"1":{"Address":"localhost:9090","Key":1},` +
	`"2":{"Address":"localhost:9091","Key":2}},"HostKey":2}`

var _ = Describe("OpenLayer", func() {
	It(
		"Should refuse storage that holds the identity of another node",
		func(ctx SpecContext) {
			store := DeferClose(storagetestutil.NewCluster()).Provision(ctx)
			Expect(store.KV.Set(ctx, []byte("aspen.cluster"), []byte(otherNode))).
				To(Succeed())
			addr := address.Address("localhost:0")
			Expect(distribution.OpenLayer(ctx, distribution.LayerConfig{
				Storage:          store,
				Transport:        transporttestutil.NewNetwork().New(addr, 1),
				AdvertiseAddress: addr,
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the data directory belongs to node 2")),
			))
		},
	)
})
