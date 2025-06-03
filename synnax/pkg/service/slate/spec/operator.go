// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package spec

import (
	"context"
	"strings"

	"github.com/samber/lo"
)

const (
	OperatorPrefix    = "comparison"
	OperatorGTESuffix = "gte"
	OperatorGTSuffix  = "gt"
	OperatorLTESuffix = "lte"
	OperatorLTSuffix  = "lt"
	OperatorEQSuffix  = "eq"
	OperatorAndSuffix = "and"
	OperatorOrSuffix  = "or"
	OperatorNotSuffix = "not"
	OperatorAddSuffix = "add"
	OperatorSubSuffix = "sub"
	OperatorMulSuffix = "mul"
	OperatorDivSuffix = "div"
)

var (
	requiresNumericOperators = []string{
		OperatorGTSuffix,
		OperatorLTSuffix,
		OperatorGTESuffix,
		OperatorLTSuffix,
		OperatorAddSuffix,
		OperatorSubSuffix,
		OperatorMulSuffix,
		OperatorDivSuffix,
	}
	numericOutputOperators = []string{
		OperatorAddSuffix,
		OperatorSubSuffix,
		OperatorMulSuffix,
		OperatorDivSuffix,
	}
	requiresBooleanOperators = []string{
		OperatorAndSuffix,
		OperatorOrSuffix,
	}
)

func containsSuffix(operator string, suffixes []string) bool {
	return lo.ContainsBy(suffixes, func(item string) bool {
		return strings.HasSuffix(item, operator)
	})
}

func operator(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if !strings.HasPrefix(n.Type, OperatorPrefix) {
		return ns, false, err
	}
	if containsSuffix(n.Type, requiresNumericOperators) {
		ns.Inputs = []Input{
			{Key: "x", AcceptsDataType: acceptsNumericDataType},
			{Key: "y", AcceptsDataType: acceptsNumericDataType},
		}
	} else if containsSuffix(n.Type, requiresBooleanOperators) {
		ns.Inputs = []Input{
			{Key: "x", AcceptsDataType: strictlyMatchDataType("uint8")},
			{Key: "y", AcceptsDataType: strictlyMatchDataType("uint8")},
		}
	} else if strings.HasSuffix(n.Type, OperatorNotSuffix) {
		ns.Inputs = []Input{{Key: "x", AcceptsDataType: strictlyMatchDataType("uint8")}}
	}
	if containsSuffix(n.Type, numericOutputOperators) {
		ns.Outputs = []Output{{Key: "value", DataType: "float64"}}
	} else {
		ns.Outputs = []Output{{Key: "value", DataType: "uint8"}}
	}
	ns.Type = n.Type
	return ns, true, nil
}
