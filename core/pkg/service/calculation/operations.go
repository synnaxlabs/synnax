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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/validate"
)

// FromChannelOperations returns the record form of ops, the operations of the channel
// API. A none operation has no record form, and a reset channel of 0 is absent. It
// returns an error wrapping validate.ErrValidation, at the position of the operation,
// for an operation whose type is not valid.
func FromChannelOperations(ops []channel.Operation) ([]Operation, error) {
	out := make([]Operation, 0, len(ops))
	for i, op := range ops {
		if err := op.Validate(); err != nil {
			return nil, validate.PathedError(err, strconv.Itoa(i))
		}
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
	return out, nil
}

// ToChannelOperations returns the channel API form of ops, which must be valid.
func ToChannelOperations(ops []Operation) []channel.Operation {
	out := make([]channel.Operation, len(ops))
	for i, op := range ops {
		var (
			typ channel.OperationType
			w   Window
		)
		switch v := op.Variant.(type) {
		case MinOperation:
			typ, w = channel.OperationTypeMin, v.Window
		case MaxOperation:
			typ, w = channel.OperationTypeMax, v.Window
		case AvgOperation:
			typ, w = channel.OperationTypeAvg, v.Window
		case DerivativeOperation:
			typ = channel.OperationTypeDerivative
		default:
			panic(fmt.Sprintf("operation of unknown type %T", op.Variant))
		}
		out[i] = channel.Operation{Type: typ, Duration: w.Duration}
		if w.Reset != nil {
			out[i].ResetChannel = *w.Reset
		}
	}
	return out
}
