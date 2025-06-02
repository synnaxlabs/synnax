// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package event

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func newSelectStatement(
	_ context.Context,
	p *plumber.Pipeline,
	cfg spec.Config,
	node spec.Node,
) (bool, error) {
	if node.Type != spec.SelectStatementType {
		return false, nil
	}
	s := &confluence.Switch[spec.Value]{
		Switch: func(ctx context.Context, value spec.Value) (address.Address, bool, error) {
			if value.Value.(uint8) == 1 {
				return "true", true, nil
			}
			return "false", true, nil
		},
	}
	plumber.SetSegment[spec.Value, spec.Value](p, address.Address(node.Key), s)
	return true, nil
}
