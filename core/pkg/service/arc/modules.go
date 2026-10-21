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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// moduleResolver resolves user modules for one analysis root. stack holds the paths
// under analysis, outermost first, so an import cycle stops instead of recursing.
type moduleResolver struct {
	svc   *Service
	tx    gorp.Tx
	stack []string
	// cycle is the first import cycle found. Every module on the chain reports it
	// instead of a bare compile failure.
	cycle error
}

var _ symbol.ModuleResolver = (*moduleResolver)(nil)

func (s *Service) newModuleResolver(tx gorp.Tx) *moduleResolver {
	return &moduleResolver{svc: s, tx: tx}
}

// ResolveModule implements symbol.ModuleResolver. The module is analyzed against a
// root sharing this resolver, so its own imports resolve the same way. That root
// becomes the module symbol.
func (r *moduleResolver) ResolveModule(
	ctx context.Context,
	path string,
) (*symbol.Symbol, error) {
	if slices.Contains(r.stack, path) {
		chain := strings.Join(r.stack, " -> ") + " -> " + path
		r.cycle = errors.Newf("import cycle:\n%s", chain)
		return nil, r.cycle
	}
	a, err := r.svc.retrieveModule(ctx, r.tx, path)
	if err != nil {
		return nil, err
	}
	r.stack = append(r.stack, path)
	defer func() { r.stack = r.stack[:len(r.stack)-1] }()
	cfg := parser.Config{AllowDashedNames: r.svc.AllowDashedNames()}
	t, diag := text.Parse(a.Text.Materialize(), cfg)
	var i ir.IR
	if diag == nil {
		i, diag = text.Analyze(ctx, t, r.svc.newRoot(r.tx, r), true, cfg)
	}
	if diag != nil && !diag.Ok() {
		if r.cycle != nil {
			return nil, r.cycle
		}
		return nil, errors.Newf("module %q does not compile", path)
	}
	mod := i.Symbols
	mod.Name, mod.Kind, mod.AST, mod.TypeMap = path, symbol.KindModule, t.AST, i.TypeMap
	return mod, nil
}

// SearchModules implements symbol.ModuleResolver.
func (r *moduleResolver) SearchModules(
	ctx context.Context,
	prefix string,
) ([]string, error) {
	paths, err := r.svc.retrieveModulePaths(ctx, r.tx)
	if err != nil {
		return nil, err
	}
	return lo.Filter(paths, func(p string, _ int) bool {
		return strings.HasPrefix(p, prefix)
	}), nil
}

// retrieveModule returns the module Arc at path, or symbol.ErrModuleNotFound.
func (s *Service) retrieveModule(
	ctx context.Context,
	tx gorp.Tx,
	path string,
) (Arc, error) {
	notFound := errors.Wrapf(symbol.ErrModuleNotFound, "unknown module %q", path)
	segments := strings.Split(path, ".")
	parent := s.group.OntologyID()
	for _, name := range segments[:len(segments)-1] {
		groups, err := s.retrieveChildren(ctx, tx, parent, ontology.ResourceTypeGroup, false)
		if err != nil {
			return Arc{}, err
		}
		g, ok := lo.Find(groups, func(r ontology.Resource) bool { return r.Name == name })
		if !ok {
			return Arc{}, notFound
		}
		parent = g.ID
	}
	arcs, err := s.retrieveChildren(ctx, tx, parent, ontology.ResourceTypeArc, true)
	if err != nil {
		return Arc{}, err
	}
	if len(arcs) == 0 {
		return Arc{}, notFound
	}
	keys, err := KeysFromOntologyIDs(ontology.ResourceIDs(arcs))
	if err != nil {
		return Arc{}, err
	}
	var entries []Arc
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&entries).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return Arc{}, err
	}
	name := segments[len(segments)-1]
	a, ok := lo.Find(entries, func(a Arc) bool {
		return a.Name == name && a.Type == TypeModule
	})
	if !ok {
		return Arc{}, notFound
	}
	return a, nil
}

// retrieveModulePaths returns the import path of every module below the Arcs group,
// sorted. A path joins the group names from the Arcs group down to the module with ".".
func (s *Service) retrieveModulePaths(
	ctx context.Context,
	tx gorp.Tx,
) ([]string, error) {
	type frame struct {
		id     ontology.ID
		prefix string
	}
	var (
		frontier = []frame{{id: s.group.OntologyID()}}
		visited  = set.New(s.group.OntologyID())
		ids      []ontology.ID
		prefixes = map[ontology.ID]string{}
	)
	for len(frontier) > 0 {
		f := frontier[0]
		frontier = frontier[1:]
		groups, err := s.retrieveChildren(ctx, tx, f.id, ontology.ResourceTypeGroup, false)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if visited.Contains(g.ID) {
				continue
			}
			visited.Add(g.ID)
			frontier = append(frontier, frame{id: g.ID, prefix: f.prefix + g.Name + "."})
		}
		arcs, err := s.retrieveChildren(ctx, tx, f.id, ontology.ResourceTypeArc, true)
		if err != nil {
			return nil, err
		}
		for _, a := range arcs {
			if _, seen := prefixes[a.ID]; seen {
				continue
			}
			prefixes[a.ID] = f.prefix
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return nil, err
	}
	var entries []Arc
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&entries).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, a := range entries {
		if a.Type != TypeModule {
			continue
		}
		paths = append(paths, prefixes[a.OntologyID()]+a.Name)
	}
	slices.Sort(paths)
	return paths, nil
}

// retrieveChildren returns the children of parent with the given type, sorted. Field
// data is skipped when excludeData is true, which leaves Name empty.
func (s *Service) retrieveChildren(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	typ ontology.ResourceType,
	excludeData bool,
) ([]ontology.Resource, error) {
	var children []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(typ).
		ExcludeFieldData(excludeData).
		Entries(&children).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	imex.SortResources(children)
	return children, nil
}
