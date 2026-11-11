// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	"go.uber.org/zap"
)

// MigrateArc lifts a v3 Arc, keeping only the body its mode selects. A blank mode, from
// files written before modes, is graph. Text is rewritten from the grammar that wrote
// inputs in braces and triggers in parens, then its mixed-type range() calls get the
// casts of CastRanges. A stored Arc carries text as a document and an exported one as
// raw, and either way the result carries a fresh document and no raw. Every Arc stored
// before v4 is an automation. Names lex with dashed names off.
func MigrateArc(ctx context.Context, old v3.Arc) (Arc, error) {
	return migrateArc(ctx, old, parser.Config{})
}

func migrateArc(ctx context.Context, old v3.Arc, cfg parser.Config) (Arc, error) {
	b := Base{Key: old.Key, Name: old.Name}
	if old.Mode != v3.ModeText {
		b.Source.Variant = GraphSource{Graph: old.Graph}
		return Arc{Variant: AutomationArc{Base: b}}, nil
	}
	t := old.Text
	raw := t.Raw
	if len(t.Doc.Inserts) > 0 {
		raw = t.Materialize().Raw
	}
	if raw != "" {
		t.Doc = text.Create(CastRanges(ctx, SwapBrackets(raw, cfg), cfg))
		t.Raw = ""
	}
	b.Source.Variant = TextSource{Text: t}
	return Arc{Variant: AutomationArc{Base: b}}, nil
}

// NewMigration returns the migration that rewrites the text of every stored Arc for the
// bracket swap and the casts of CastRanges. cfg must match the parser config the Core
// that stored the Arcs ran with, since a dashed name lexes as subtraction without it.
func NewMigration(cfg parser.Config) gorp.Migration {
	return migrate.NewInstrumentedEntry(
		"v61_swap_brackets",
		func(ctx context.Context, old v3.Arc, ins alamos.Instrumentation) (Arc, error) {
			a, err := migrateArc(ctx, old, cfg)
			if err != nil {
				return Arc{}, err
			}
			src, ok := a.Base().Source.Variant.(TextSource)
			if !ok {
				return a, nil
			}
			if _, diag := parser.Parse(src.Text.Materialize().Raw, cfg); diag != nil {
				ins.L.Warn(
					"left the range() calls of an Arc that does not parse unchanged",
					zap.Stringer("arc", old.Key),
					zap.Error(diag),
				)
			}
			return a, nil
		},
	)
}

// NewRootMigration returns the migration that puts every stored Arc in no directory
// into the directory root.
func NewRootMigration(root uuid.UUID) gorp.Migration {
	return migrate.NewEntry(
		"v61_arcs_root_directory",
		func(_ context.Context, a Arc) (Arc, error) {
			if b := a.Base(); b.Directory == nil {
				b.Directory = &root
				a.SetBase(b)
			}
			return a, nil
		},
	)
}
