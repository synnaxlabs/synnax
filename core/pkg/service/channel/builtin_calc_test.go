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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Built-in channel named like a calculated channel", func() {
	It("Should not make a calculated user channel a built-in", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		cfg, _ := serviceConfig(ctx, node, node.DB)
		var calcs *calculation.Service
		cfg.Fills = func(
			ctx context.Context,
			tx gorp.Tx,
			keys ...channel.Key,
		) (map[channel.Key]channel.Fill, error) {
			return calcs.Fills(ctx, tx, keys...)
		}
		svc := MustOpen(channel.OpenService(ctx, cfg))
		calcs = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: svc,
		}))
		name := UniqueChannelName()
		// A user made this calculation before a Core version declared a built-in of its
		// name.
		calc := calculation.Calculation{Expression: "return 1.0 + 2.0"}
		var output channel.Channel
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			created, err := calcs.NewWriter(tx).CreateOutput(ctx, &calc, name, nil)
			if err == nil {
				output = created[0]
			}
			return err
		})).To(Succeed())
		builtIn := channel.Channel{Name: name, DataType: output.DataType, Virtual: true}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		Expect(builtIn.Key()).ToNot(Equal(output.Key()))
		Expect(svc.IsBuiltIn(output.Key())).To(BeFalse())
	})
})
