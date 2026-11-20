// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// FromChannelOperations returns the calculation form of the operations of a channel. A
// none operation has no calculation form, and a reset channel of 0 is absent. It panics
// on an operation whose type is not valid.
func FromChannelOperations(ops []channel.Operation) []Operation {
	out := make([]Operation, 0, len(ops))
	for _, op := range ops {
		w := Window{Duration: op.Duration}
		if op.ResetChannel != 0 {
			w.Reset = new(op.ResetChannel)
		}
		var variant OperationVariant
		switch op.Type {
		case channel.OperationTypeMin:
			variant = MinOperation{Window: w}
		case channel.OperationTypeMax:
			variant = MaxOperation{Window: w}
		case channel.OperationTypeAvg:
			variant = AvgOperation{Window: w}
		case channel.OperationTypeDerivative:
			variant = DerivativeOperation{}
		case channel.OperationTypeNone:
			continue
		default:
			panic(fmt.Sprintf("operation of unknown type %q", op.Type))
		}
		out = append(out, Operation{Variant: variant})
	}
	return out
}
