// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeLog }

// Match reports whether body is a legacy Console log state, which persists the channels
// as an array. The markers are frozen historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, ok := body["channels"].([]any)
	return ok
}

// Import decodes env into a Log created under opts.Parent, which must be a project. A
// group must be in the parent project. The key on the wire is discarded so every import
// mints a new resource. The log drops each channel the Core does not hold. An unknown
// envelope version is a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (resource.ID, error) {
	proj, err := project.KeyFromResourceID(opts.Parent)
	if err != nil {
		return resource.ID{}, validate.PathedError(err, "parent")
	}
	if err = project.ValidateGroup(
		ctx, tx, proj, opts.Group, resource.TypeLog,
	); err != nil {
		return resource.ID{}, err
	}
	l, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return resource.ID{}, err
	}
	l.Group, l.Project = nil, nil
	if _, err = s.table.DropDangling(ctx, tx, &l); err != nil {
		return resource.ID{}, err
	}
	l.Group, l.Project = opts.Group, &proj
	if err = s.NewWriter(tx).Create(ctx, &l); err != nil {
		return resource.ID{}, err
	}
	return l.ResourceID(), nil
}
