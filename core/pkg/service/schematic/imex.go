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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
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
// discarded so every import mints a new resource. The schematic drops each named color
// and each enumeration that the Core does not hold, so a select or state indicator that
// loses its enumeration draws its own options. A node whose custom symbol the Core does
// not hold draws a missing symbol with the same key, and every other node drops the
// bindings that its symbol orphans. An unknown envelope version is a path-scoped
// validation error.
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
		sch.Group, sch.Project = nil, nil
		if err = s.fitNodes(ctx, tx, &sch); err != nil {
			return resource.ID{}, err
		}
		if _, err = s.table.DropDangling(ctx, tx, &sch); err != nil {
			return resource.ID{}, err
		}
		sch.Group, sch.Project = opts.Group, &proj
		if err = s.NewWriter(tx).Create(ctx, &sch); err != nil {
			return resource.ID{}, err
		}
		return ResourceID(sch.Key), nil
	}, nil
}

// fitNodes makes each node of sch whose custom symbol tx lacks draw a missing symbol
// with the same key, so DropDangling keeps the node, and drops the bindings that the
// symbol of each other node orphans.
func (s *Service) fitNodes(ctx context.Context, tx gorp.Tx, sch *Schematic) error {
	held, err := s.NewWriter(tx).customSymbols(ctx, customKeys(*sch))
	if err != nil {
		return err
	}
	for i, n := range sch.Nodes {
		cfg, ok := n.Config.Variant.(SymbolNodeConfig)
		if !ok {
			continue
		}
		switch ref := cfg.Spec.Variant.(type) {
		case symbol.BuiltinRef:
			if b, ok := s.builtins[ref.Name]; ok {
				cfg, _ = dropOrphans(cfg, b.States, b.Commands)
			}
		case symbol.CustomRef:
			if sym, ok := held[ref.Key]; ok {
				cfg, _ = dropOrphans(cfg, sym.Data.States, sym.Data.Commands)
			} else {
				cfg.Spec = symbol.Ref{Variant: symbol.MissingRef{Key: ref.Key.String()}}
			}
		}
		sch.Nodes[i].Config.Variant = cfg
	}
	return nil
}

// customKeys returns the keys of the custom symbols that the nodes of s draw.
func customKeys(s Schematic) []symbol.Key {
	var keys []symbol.Key
	for _, n := range s.Nodes {
		if cfg, ok := n.Config.Variant.(SymbolNodeConfig); ok {
			if ref, ok := cfg.Spec.Variant.(symbol.CustomRef); ok {
				keys = append(keys, ref.Key)
			}
		}
	}
	return lo.Uniq(keys)
}
