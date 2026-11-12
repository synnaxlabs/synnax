// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeTable }

// Match reports whether body is a legacy Console table state, which persists the
// structural model inline under layout and cells. The markers are frozen historical
// file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasLayout := body["layout"]
	_, hasCells := body["cells"]
	return hasLayout && hasCells
}

// Decode decodes env into a Table that its Write creates under opts.Parent, which must
// be a project. A group must be in the parent project. The key on the wire is discarded
// so every import mints a new resource. An unknown envelope version is a path-scoped
// validation error.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	t, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		opts imex.ImportOptions,
	) (resource.ID, error) {
		proj, err := project.KeyFromResourceID(opts.Parent)
		if err != nil {
			return resource.ID{}, validate.PathedError(err, "parent")
		}
		if err = project.ValidateGroup(
			ctx, tx, proj, opts.Group, resource.TypeTable,
		); err != nil {
			return resource.ID{}, err
		}
		t.Group, t.Project, t.Range = opts.Group, &proj, nil
		if err = s.NewWriter(tx).Create(ctx, &t); err != nil {
			return resource.ID{}, err
		}
		return ResourceID(t.Key), nil
	}, nil
}
