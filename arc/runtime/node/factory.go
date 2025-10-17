// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Config struct {
	alamos.Instrumentation
	Node   ir.Node
	State  *state.Node
	Module module.Module
}

type Factory interface {
	Create(ctx context.Context, cfg Config) (Node, error)
}

type MultiFactory []Factory

func (mf MultiFactory) Create(ctx context.Context, cfg Config) (Node, error) {
	for _, f := range mf {
		n, err := f.Create(ctx, cfg)
		if err != nil {
			if errors.Is(err, query.NotFound) {
				continue
			}
			return nil, err
		}
		return n, nil
	}
	return nil, query.NotFound
}
