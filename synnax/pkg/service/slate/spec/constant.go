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
)

func constant(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "constant" {
		return ns, false, err
	}
	fields := map[string]schema.Field{
		"data_type": {
			Type: schema.String,
		},
	}
	dt, ok := schema.Get[string](schema.Resource{Data: n.Data}, "data_type")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "data_type",
			Message: "invalid data type",
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
	ns.Type = "constant"
	return ns, true, nil
}
