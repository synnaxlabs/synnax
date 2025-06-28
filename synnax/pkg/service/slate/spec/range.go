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

	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/zyn"
)

const RangeCreateType = "range.create"

type RangeCreateConfig struct {
	Range ranger.Range `json:"range" msgpack:"range"`
}

var rangeCreateConfigZ = zyn.Object(map[string]zyn.Schema{
	"range": ranger.RangeZ,
})

func (r *RangeCreateConfig) Parse(data any) error {
	return rangeCreateConfigZ.Parse(data, r)
}

func rangeCreator(_ context.Context, _ Config, n Node) (NodeSchema, bool, error) {
	var ns NodeSchema
	if n.Type != RangeCreateType {
		return ns, false, nil
	}
	c := &RangeCreateConfig{}
	if err := c.Parse(n.Config); err != nil {
		return ns, true, err
	}
	ns.Inputs = []Input{
		{Key: string(ranger.ToDo), AcceptsDataType: zyn.AnyDataTypeSchema},
		{Key: string(ranger.InProgress), AcceptsDataType: zyn.AnyDataTypeSchema},
		{Key: string(ranger.Completed), AcceptsDataType: zyn.AnyDataTypeSchema},
	}
	ns.Type = RangeCreateType
	return ns, true, nil
}
