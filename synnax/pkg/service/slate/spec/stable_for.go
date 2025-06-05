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

const StableForType = "stable_for"

func stableFor(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != StableForType {
		return ns, false, err
	}
	_, ok = schema.Get[float64](schema.Resource{Data: n.Data}, "duration")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "duration",
			Message: "invalid duration",
		})
	}
	ns.Inputs = []Input{
		{
			Key:             "input",
			AcceptsDataType: acceptsNumericDataType,
		},
	}
	ns.Outputs = []Output{
		{
			Key:      "output",
			DataType: "uint8",
		},
	}
	ns.Data = map[string]schema.Field{
		"duration": {Type: schema.Float64},
	}
	ns.Type = StableForType
	return ns, true, nil
}
