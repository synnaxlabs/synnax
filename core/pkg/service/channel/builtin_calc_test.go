// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/gorp"
	distnode "github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Built-in channel named like a calculated channel", func() {
	It("Should not make the output of a calculation a built-in", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		name := UniqueChannelName()
		// A user made this output before a Core version declared a built-in of its
		// name.
		output := channel.Channel{
			Name:        name,
			DataType:    telem.Float64T,
			Virtual:     true,
			Leaseholder: distnode.KeyFree,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &output, channel.FreeIndex())
		})).To(Succeed())
		builtIn := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		Expect(builtIn.Key()).ToNot(Equal(output.Key()))
		Expect(svc.IsBuiltIn(output.Key())).To(BeFalse())
	})
})
