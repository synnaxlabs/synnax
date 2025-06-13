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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

const ConstantType = "constant"

type ConstantConfig struct {
	dataType zyn.DataType
}

var configZ = zyn.Object(map[string]zyn.Z{
	"data_type": zyn.PrimitiveTypeZ,
})

func (c *ConstantConfig) Parse(data any) error { return configZ.Parse(data, c) }

func constant(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != ConstantType {
		return ns, false, err
	}
	c := &ConstantConfig{}
	if err = c.Parse(n.Data); err != nil {
		return ns, false, err
	}

	dt, ok := schema.Get[string](res, "data_type")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "data_type",
			Message: "invalid data type",
		})
	}
	_, ok = schema.Get[any](res, "value")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "value",
			Message: "invalid value",
		})
	}
	fields["value"] = schema.Field{Type: schema.FieldType(dt)}
	ns.Outputs = []Output{
		{
			Key:      "value",
			DataType: dt,
		},
	}
	ns.Data = fields
	ns.Type = ConstantType
	return ns, true, nil
}
