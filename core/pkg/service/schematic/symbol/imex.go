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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

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

// Import decodes env into a Symbol created in the group opts.Parent identifies.
// The key on the wire is discarded so every import mints a new resource. An unknown
// envelope version is a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (resource.ID, error) {
	g, err := group.KeyFromResourceID(opts.Parent)
	if err != nil {
		return resource.ID{}, validate.PathedError(err, "parent")
	}
	sym, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return resource.ID{}, err
	}
	sym.Group = &g
	if err = s.NewWriter(tx).Create(ctx, &sym); err != nil {
		return resource.ID{}, err
	}
	return sym.ResourceID(), nil
}
