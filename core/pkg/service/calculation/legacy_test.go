// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Legacy", func() {
	Describe("FromChannelOperations", func() {
		It("Should give each operation its calculation form, without none", func() {
			reset := channel.Key(7)
			Expect(calculation.FromChannelOperations([]channel.Operation{
				{Type: channel.OperationTypeAvg, Duration: telem.Second},
				{Type: channel.OperationTypeNone},
				{Type: channel.OperationTypeMax, ResetChannel: reset},
				{Type: channel.OperationTypeDerivative},
			})).To(Equal([]calculation.Operation{
				{Variant: calculation.AvgOperation{
					Window: calculation.Window{Duration: telem.Second},
				}},
				{Variant: calculation.MaxOperation{
					Window: calculation.Window{Reset: new(reset)},
				}},
				{Variant: calculation.DerivativeOperation{}},
			}))
		})

		It("Should fail at the position of an operation of an unknown type", func() {
			Expect(calculation.FromChannelOperations([]channel.Operation{
				{Type: channel.OperationTypeMin},
				{Type: "sum"},
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("1.type")),
			))
		})
	})

	Describe("ChannelOperations", func() {
		It("Should return the channel form that FromChannelOperations reads", func() {
			ops := []channel.Operation{
				{Type: channel.OperationTypeMin, Duration: telem.Second},
				{Type: channel.OperationTypeAvg, ResetChannel: channel.Key(9)},
				{Type: channel.OperationTypeDerivative},
			}
			calc := MustSucceed(calculation.FromChannelOperations(ops))
			Expect(calculation.ChannelOperations(calc)).To(Equal(ops))
		})
	})

	Describe("EditsCalculation", func() {
		set := func(path ...string) channel.Action {
			return channel.NewSetAction(channel.SetPayload{Path: path, Value: 1})
		}
		It("Should report an edit of the expression or the operations", func() {
			Expect(calculation.EditsCalculation(set("group"), set("expression"))).
				To(BeTrue())
			Expect(calculation.EditsCalculation(set("operations", "0"))).To(BeTrue())
		})

		It("Should not report an edit of other members or a rename", func() {
			Expect(calculation.EditsCalculation(
				set("group"),
				channel.NewRenameAction(channel.RenamePayload{Name: "renamed"}),
			)).To(BeFalse())
		})
	})
})
