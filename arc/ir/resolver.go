// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type SymbolResolver interface {
	Resolve(ctx context.Context, name string) (Symbol, error)
}

type MapResolver map[string]Symbol

var _ SymbolResolver = (*MapResolver)(nil)

func (m MapResolver) Resolve(_ context.Context, name string) (Symbol, error) {
	if s, ok := m[name]; ok {
		return s, nil
	}
	return Symbol{}, errors.Wrapf(query.NotFound, "symbol %s not found", name)
}

type CompoundResolver []SymbolResolver

func (c CompoundResolver) Resolve(ctx context.Context, name string) (Symbol, error) {
	var (
		symbol Symbol
		err    error
	)
	for _, s := range c {
		symbol, err = s.Resolve(ctx, name)
		if err == nil {
			return symbol, nil
		}
	}
	return symbol, err
}
