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

	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

const StableForType = "stable_for"

type StableForConfig struct {
	Duration telem.TimeSpan
}

func (s *StableForConfig) Parse(data any) error {
	return stableForConfigZ.Parse(data, s)
}

var stableForConfigZ = zyn.Object(map[string]zyn.Schema{
	"duration": zyn.Int64().Coerce(),
})

func stableFor(_ context.Context, _ Config, n Node) (NodeSchema, bool, error) {
	var ns NodeSchema
	if n.Type != StableForType {
		return ns, false, nil
	}
	if err := stableForConfigZ.Validate(n); err != nil {
		return ns, true, err
	}
	ns.Inputs = []Input{{Key: "input", AcceptsDataType: zyn.NumericTypeSchema}}
	ns.Outputs = []Output{{Key: "output", DataType: zyn.BoolT}}
	ns.Type = StableForType
	return ns, true, nil
}
