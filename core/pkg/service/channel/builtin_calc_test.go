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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Built-in channel named like a calculated channel", func() {
	It("Should not make a calculated user channel a built-in", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		name := UniqueChannelName()
		// A user made this channel before a Core version declared a built-in of its
		// name.
		calc := channel.Channel{Name: name, Expression: "return 1.0 + 2.0"}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &calc)
		})).To(Succeed())
		GinkgoWriter.Printf(
			"user channel: data type %s, virtual %t, calculated %t\n",
			calc.DataType,
			calc.Virtual,
			calc.IsCalculated(),
		)
		builtIn := channel.Channel{Name: name, DataType: calc.DataType, Virtual: true}
		err := node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})
		GinkgoWriter.Printf("Declare: %v\n", err)
		Expect(builtIn.IsCalculated()).To(BeFalse())
		Expect(svc.IsBuiltIn(calc.Key())).To(BeFalse())
	})
})
