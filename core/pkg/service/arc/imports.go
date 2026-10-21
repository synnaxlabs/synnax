// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// OntologyRelationshipTypeImports records that an Arc imports a module: From imports
// To. The edges are rewritten from the import statements on every save.
const OntologyRelationshipTypeImports ontology.RelationshipType = "imports"

// ImportersTraverser traverses from a module to the Arcs that import it.
var ImportersTraverser = ontology.IncomingTraverser(OntologyRelationshipTypeImports)

// syncImports rewrites the outgoing imports edges of a from its import statements. A
// path with no module leaves no edge. An edge that would close a cycle is skipped, the
// compiler reports the cycle. A source that does not parse keeps its edges.
func (s *Service) syncImports(ctx context.Context, tx gorp.Tx, a Arc) error {
	if a.Mode != ModeText {
		return nil
	}
	cfg := parser.Config{AllowDashedNames: s.AllowDashedNames()}
	t, diag := text.Parse(a.Text.Materialize(), cfg)
	if diag != nil {
		return nil
	}
	w := s.cfg.Ontology.NewWriter(tx)
	if err := w.DeleteOutgoingRelationshipsOfType(
		ctx, a.OntologyID(), OntologyRelationshipTypeImports,
	); err != nil {
		return err
	}
	for _, imp := range parser.Imports(t.AST) {
		mod, err := s.retrieveModule(ctx, tx, imp.Path)
		if errors.Is(err, symbol.ErrModuleNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if err = w.DefineRelationships(
			ctx, a.OntologyID(), OntologyRelationshipTypeImports, mod.OntologyID(),
		); err != nil && !errors.Is(err, graph.ErrCyclicDependency) {
			return err
		}
	}
	return nil
}

// Importers returns the keys of every Arc that imports the Arc with the given key,
// directly or through another module.
func (s *Service) Importers(ctx context.Context, tx gorp.Tx, key Key) ([]Key, error) {
	var (
		frontier = []ontology.ID{OntologyID(key)}
		visited  = set.New(OntologyID(key))
		ids      []ontology.ID
	)
	for len(frontier) > 0 {
		var res []ontology.Resource
		if err := s.cfg.Ontology.NewRetrieve().
			WhereIDs(frontier...).
			TraverseTo(ImportersTraverser).
			ExcludeFieldData(true).
			Entries(&res).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return nil, err
		}
		var next []ontology.ID
		for _, r := range res {
			if visited.Contains(r.ID) {
				continue
			}
			visited.Add(r.ID)
			next = append(next, r.ID)
		}
		ids = append(ids, next...)
		frontier = next
	}
	return KeysFromOntologyIDs(ids)
}

// resyncImporters schedules a resync of every Arc that imports the Arc with the given
// key.
func (s *Service) resyncImporters(ctx context.Context, tx gorp.Tx, key Key) error {
	keys, err := s.Importers(ctx, tx, key)
	if err != nil {
		return err
	}
	for _, k := range keys {
		s.taskSync.Trigger(k)
	}
	return nil
}

// resyncAll schedules a resync of every Arc. A new module may satisfy imports that had
// no module to point an edge at.
func (s *Service) resyncAll(ctx context.Context, tx gorp.Tx) error {
	var arcs []Arc
	if err := s.table.NewRetrieve().
		Entries(&arcs).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		return err
	}
	for _, a := range arcs {
		s.taskSync.Trigger(a.Key)
	}
	return nil
}
