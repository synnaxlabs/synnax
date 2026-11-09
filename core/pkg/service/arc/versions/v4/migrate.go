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

	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
)

// MigrateArc lifts a v3 Arc, keeping only the body its mode selects. A blank mode, from
// files written before modes, is graph. Text is rewritten from the grammar that wrote
// inputs in braces and triggers in parens; a stored Arc carries it as a document and an
// exported one as raw, and either way the result carries a fresh document and no raw.
// Every Arc stored before v4 is an automation.
func MigrateArc(_ context.Context, old v3.Arc) (Arc, error) {
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
		t.Doc = text.Create(SwapBrackets(raw))
		t.Raw = ""
	}
	b.Source.Variant = TextSource{Text: t}
	return Arc{Variant: AutomationArc{Base: b}}, nil
}

// Migration rewrites the text of every stored Arc for the bracket swap.
var Migration = migrate.NewEntry("v61_swap_brackets", MigrateArc)

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
