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

	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/zyn"
)

const StatusChangeType = "status.change"

type StatusChangerConfig struct {
	Message string
	Variant status.Variant
}

var statusChangerConfigZ = zyn.Object(map[string]zyn.Schema{
	"message": zyn.String(),
	"variant": status.VariantZ,
})

func (c *StatusChangerConfig) Parse(data any) error {
	return statusChangerConfigZ.Parse(data, c)
}

func createStatusChanger(_ context.Context, _ Config, n Node) (NodeSchema, bool, error) {
	var ns NodeSchema
	if n.Type != StatusChangeType {
		return ns, false, nil
	}
	if err := statusChangerConfigZ.Validate(n.Config); err != nil {
		return ns, false, err
	}
	ns.Inputs = []Input{{
		Key:             "value",
		AcceptsDataType: zyn.PrimitiveTypeSchema,
	}}
	ns.Outputs = []Output{}
	ns.Type = StatusChangeType
	return ns, true, nil
}
