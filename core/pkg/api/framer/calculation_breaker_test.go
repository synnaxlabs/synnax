// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation breaker", func() {
	virtual := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	It(
		"Should deny a read of a calculation whose record, edited through the "+
			"calculation API, reads a channel the subject cannot read",
		func(ctx SpecContext) {
			visible, hidden := virtual(ctx), virtual(ctx)
			output := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       UniqueChannelName(),
				Expression: fmt.Sprintf("return %s * 2", visible.Name),
			}))
			c := MustSucceed(creator.CalculationOf(ctx, output.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Calculation.NewWriter(tx).Apply(
					ctx,
					c.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: fmt.Sprintf("return %s * 2", hidden.Name),
					}),
				)
			})).To(Succeed())
			u := createUserGranted(
				ctx,
				svcframer.ResourceIDs(channel.Keys{output.Key(), visible.Key()})...,
			)
			_, err := apiSvc.Read(
				subjectCtx(ctx, u.ResourceID()),
				apiframer.ReadRequest{
					Keys:   channel.Keys{output.Key()},
					Bounds: telem.TimeRangeMax,
				},
			)
			Expect(err).To(MatchError(access.ErrDenied))
		},
	)

	It(
		"Should allow a read of a channel whose calculation is deleted without "+
			"access to the channels the calculation read",
		func(ctx SpecContext) {
			input := virtual(ctx)
			output := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       UniqueChannelName(),
				Expression: fmt.Sprintf("return %s * 2", input.Name),
			}))
			c := MustSucceed(creator.CalculationOf(ctx, output.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Calculation.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
			u := createUserGranted(
				ctx,
				svcframer.ResourceIDs(channel.Keys{output.Key()})...,
			)
			_, err := apiSvc.Read(
				subjectCtx(ctx, u.ResourceID()),
				apiframer.ReadRequest{
					Keys:   channel.Keys{output.Key()},
					Bounds: telem.TimeRangeMax,
				},
			)
			Expect(err).ToNot(
				HaveOccurred(),
				"output %s, old input %s",
				output.Key(),
				input.Key(),
			)
		},
	)
})
