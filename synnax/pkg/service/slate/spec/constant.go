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

const ConstantType = "constant"

type ConstantConfig struct {
	DataType zyn.DataType
	Value    any
}

var configZ = zyn.Object(map[string]zyn.Schema{
	"data_type": zyn.PrimitiveTypeSchema,
	"value":     zyn.Primitive(),
})

func (c *ConstantConfig) Parse(data any) error { return configZ.Parse(data, c) }

func constant(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != ConstantType {
		return ns, false, err
	}
	c := &ConstantConfig{}
	if err = c.Parse(n.Config); err != nil {
		return ns, true, err
	}
	ns.Type = ConstantType
	ns.Outputs = []Output{{
		Key:      "value",
		DataType: c.DataType,
	}}
	return ns, true, nil
}
