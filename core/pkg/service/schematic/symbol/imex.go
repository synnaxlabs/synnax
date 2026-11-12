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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
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

// Decode decodes env into a Symbol that its Write creates in opts.Group. opts.Parent is
// ignored: a symbol has no parent. The key on the wire is discarded so every import
// mints a new resource. An unknown envelope version is a path-scoped validation error.
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
		sym.Group = opts.Group
		if err := s.NewWriter(tx).Create(ctx, &sym); err != nil {
			return resource.ID{}, err
		}
		return sym.ResourceID(), nil
	}, nil
}
