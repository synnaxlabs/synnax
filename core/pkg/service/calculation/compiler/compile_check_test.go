// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// configWithoutSTL returns the config of a new calculation over a new channel named
// name, with no standard library symbols.
func configWithoutSTL(ctx SpecContext, name string) compiler.Config {
	GinkgoHelper()
	base := channel.Channel{Name: name + "_base", DataType: telem.Int64T, Virtual: true}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &base)
	})).To(Succeed())
	calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
		Name:       name,
		DataType:   telem.Int64T,
		Expression: "return " + base.Name + " * 2",
	}))
	return compiler.Config{
		ChannelService: channelSvc,
		Channel:        calc,
		Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
	}
}

var _ = Describe("Compile check", func() {
	It("Should require the Arc standard library symbols", func(ctx SpecContext) {
		Expect(compiler.Compile(ctx, db, configWithoutSTL(ctx, "stl_check"))).
			Error().To(MatchError(ContainSubstring("stl: must be non-empty")))
	})

	It("Should require the Arc standard library symbols to pre-process", func(
		ctx SpecContext,
	) {
		Expect(compiler.PreProcess(ctx, db, configWithoutSTL(ctx, "stl_pre_check"))).
			Error().To(MatchError(ContainSubstring("stl: must be non-empty")))
	})
})
