// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/errors"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeSchematicSymbol }

// Match reports whether body is a legacy Console symbol file, which nests the spec in a
// data object with an inline svg. The marker is a frozen historical file shape.
func (*Service) Match(body map[string]any) bool {
	data, ok := body["data"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = data["svg"]
	return ok
}

// Decode decodes env into a Symbol that its Write creates in opts.Group under the
// bundled key. When the Core already has a symbol with that key, Write reuses it as it
// is, in its group, and returns an error wrapping access.ErrDenied if opts.Retrievable
// does not hold it. opts.Parent is ignored: a symbol has no parent. A state keeps an
// enumeration that the Core has and clears one that it lacks. An unknown envelope
// version is a path-scoped validation error.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	sym, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		opts imex.ImportOptions,
	) (resource.ID, error) {
		reused, err := s.reused(ctx, tx, sym.Key, opts.Retrievable)
		if err != nil || reused {
			return sym.ResourceID(), err
		}
		if _, err := s.table.DropDangling(ctx, tx, &sym); err != nil {
			return resource.ID{}, err
		}
		sym.Group = opts.Group
		if err := s.NewWriter(tx).Create(ctx, &sym); err != nil {
			return resource.ID{}, err
		}
		return sym.ResourceID(), nil
	}, nil
}

// reused reports whether the Core has a symbol with key, which an import reuses. It
// returns an error wrapping access.ErrDenied when retrievable does not hold that symbol.
func (s *Service) reused(
	ctx context.Context,
	tx gorp.Tx,
	key Key,
	retrievable func(resource.ID) bool,
) (bool, error) {
	if key == uuid.Nil() {
		return false, nil
	}
	exists, err := s.NewRetrieve().Where(MatchKeys(key)).Exists(ctx, tx)
	if err != nil || !exists {
		return false, err
	}
	if retrievable == nil || !retrievable(ResourceID(key)) {
		return false, errors.Wrapf(access.ErrDenied, "cannot retrieve symbol %s", key)
	}
	return true, nil
}
