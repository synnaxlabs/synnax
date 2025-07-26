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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

const (
	OperatorPrefix    = "operator"
	OperatorGTESuffix = "ge"
	OperatorGTSuffix  = "gt"
	OperatorLTESuffix = "le"
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
		return strings.HasSuffix(operator, item)
	})
}

func operator(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if !strings.HasPrefix(n.Type, OperatorPrefix) {
		return ns, false, err
	}
	if containsSuffix(n.Type, requiresNumericOperators) {
		ns.Inputs = []Input{
			{Key: "x", AcceptsDataType: zyn.NumericTypeSchema},
			{Key: "y", AcceptsDataType: zyn.NumericTypeSchema},
		}
	} else if containsSuffix(n.Type, requiresBooleanOperators) {
		ns.Inputs = []Input{
			{Key: "x", AcceptsDataType: zyn.BoolTypeSchema},
			{Key: "y", AcceptsDataType: zyn.BoolTypeSchema},
		}
	} else if strings.HasSuffix(n.Type, OperatorNotSuffix) {
		ns.Inputs = []Input{{Key: "x", AcceptsDataType: zyn.BoolTypeSchema}}
	} else {
		return ns, false, errors.Wrapf(validate.Error, "operator %s not supported", n.Type)
	}
	if containsSuffix(n.Type, numericOutputOperators) {
		ns.Outputs = []Output{{Key: "value", DataType: zyn.Float64T}}
	} else {
		ns.Outputs = []Output{{Key: "value", DataType: zyn.BoolT}}
	}
	ns.Type = n.Type
	return ns, true, nil
}
