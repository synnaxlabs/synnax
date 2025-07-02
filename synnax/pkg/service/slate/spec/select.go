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

	"github.com/synnaxlabs/x/zyn"
)

const SelectStatementType = "select"

func selectStatement(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != SelectStatementType {
		return ns, false, err
	}
	ns.Inputs = []Input{
		{
			Key:             "value",
			AcceptsDataType: zyn.Enum(zyn.BoolT, zyn.Uint8T),
		},
	}
	ns.Outputs = []Output{
		{Key: "true", DataType: zyn.BoolT},
		{Key: "false", DataType: zyn.BoolT},
	}
	ns.Type = SelectStatementType
	return ns, true, nil
}
