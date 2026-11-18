// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeSchematic }

// Match reports whether body is a legacy Console schematic state, which persists the
// document inline under nodes and props. The markers are frozen historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasNodes := body["nodes"]
	_, hasProps := body["props"]
	return hasNodes && hasProps
}

// Decode decodes env into a Schematic that its Write creates under opts.Parent, which
// must be a project. A group must be in the parent project. The key on the wire is
// discarded so every import mints a new resource. A select or state indicator keeps an
// enumeration that the Core has and clears one that it lacks, so it draws its own
// options. An unknown envelope version is a path-scoped validation error.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	sch, err := versions.DecodeImExEnvelope(ctx, env)
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
			ctx, tx, proj, opts.Group, resource.TypeSchematic,
		); err != nil {
			return resource.ID{}, err
		}
		sch.Group, sch.Project = opts.Group, &proj
		if err = s.clearMissingEnumerations(ctx, tx, &sch); err != nil {
			return resource.ID{}, err
		}
		if err = s.NewWriter(tx).Create(ctx, &sch); err != nil {
			return resource.ID{}, err
		}
		return ResourceID(sch.Key), nil
	}, nil
}

func (s *Service) clearMissingEnumerations(
	ctx context.Context,
	tx gorp.Tx,
	sch *Schematic,
) error {
	var keys []enumeration.Key
	for _, n := range sch.Nodes {
		switch cfg := n.Config.Variant.(type) {
		case SelectNodeConfig:
			if cfg.Enumeration != nil {
				keys = append(keys, *cfg.Enumeration)
			}
		case StateIndicatorNodeConfig:
			if cfg.Enumeration != nil {
				keys = append(keys, *cfg.Enumeration)
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	var found []enumeration.Enumeration
	if err := s.cfg.Enumeration.NewRetrieve().
		Where(enumeration.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, tx); err != nil {
		return err
	}
	stored := make(set.Set[enumeration.Key], len(found))
	for _, e := range found {
		stored.Add(e.Key)
	}
	keep := func(k *enumeration.Key) *enumeration.Key {
		if k == nil || !stored.Contains(*k) {
			return nil
		}
		return k
	}
	for i, n := range sch.Nodes {
		switch cfg := n.Config.Variant.(type) {
		case SelectNodeConfig:
			cfg.Enumeration = keep(cfg.Enumeration)
			sch.Nodes[i].Config.Variant = cfg
		case StateIndicatorNodeConfig:
			cfg.Enumeration = keep(cfg.Enumeration)
			sch.Nodes[i].Config.Variant = cfg
		}
	}
	return nil
}
