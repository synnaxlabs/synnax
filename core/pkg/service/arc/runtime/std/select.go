// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/x/maps"
)

var symbolSelect = ir.Symbol{
	Name: "select",
	Kind: ir.KindStage,
	Type: ir.Stage{Config: maps.Ordered[string, ir.Type]{}},
}

type selectStage struct{ base }

func (s *selectStage) Next(ctx context.Context, value stage.Value) {
	if value.Value == 0 {
		value.Param = "true"
	} else {
		value.Param = "false"
	}
	s.Next(ctx, value)
}
