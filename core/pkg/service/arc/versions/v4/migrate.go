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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
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

// MigrationConfig is the configuration for NewRootMigration.
type MigrationConfig struct {
	// Ontology holds the parent relationships of stored Arcs.
	Ontology *ontology.Ontology
	// Owner writes the parent relationships of Arcs.
	Owner *ontology.Owner
	// Root is the directory that holds every Arc.
	Root resource.ID
}

// NewRootMigration returns the migration that parents every stored Arc without a
// directory under the root directory.
func NewRootMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v61_arcs_directory",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
			iter, err := gorp.WrapReader[Key, Arc](tx).OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			var orphans []resource.ID
			for iter.First(); iter.Valid(); iter.Next() {
				a := iter.Value(ctx)
				if err = iter.Error(); err != nil {
					return err
				}
				id := a.ResourceID()
				var parents []ontology.Resource
				// The Arc ontology service is not registered yet, so field data must
				// stay excluded.
				if err = cfg.Ontology.NewRetrieve().
					WhereIDs(id).
					TraverseTo(ontology.ParentsTraverser).
					WhereTypes(resource.TypeArcDirectory).
					ExcludeFieldData(true).
					Entries(&parents).
					Exec(ctx, tx); err != nil {
					return err
				}
				if len(parents) == 0 {
					orphans = append(orphans, id)
				}
			}
			if len(orphans) == 0 {
				return nil
			}
			return cfg.Owner.NewWriter(tx).DefineRelationships(
				ctx,
				cfg.Root,
				ontology.RelationshipTypeParentOf,
				orphans...,
			)
		},
	)
}
