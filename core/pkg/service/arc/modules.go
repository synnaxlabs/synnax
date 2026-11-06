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
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
)

// moduleSource returns a loader source that reads the text of the module at an
// import path, one directory below Arcs per segment.
func (s *Service) moduleSource(
	tx gorp.Tx,
) func(context.Context, string) (arctext.Text, error) {
	return func(ctx context.Context, path string) (arctext.Text, error) {
		return s.retrieveModuleText(ctx, tx, path)
	}
}

func (s *Service) retrieveModuleText(
	ctx context.Context,
	tx gorp.Tx,
	path string,
) (arctext.Text, error) {
	notFound := errors.Wrapf(symbol.ErrModuleNotFound, "unknown module %q", path)
	segments := strings.Split(path, ".")
	parent := directory.OntologyID(s.root)
	for _, name := range segments[:len(segments)-1] {
		dir, ok, err := s.childNamed(
			ctx, tx, parent, ontology.ResourceTypeArcDirectory, name,
		)
		if err != nil {
			return arctext.Text{}, err
		}
		if !ok {
			return arctext.Text{}, notFound
		}
		parent = dir
	}
	id, ok, err := s.childNamed(
		ctx, tx, parent, ontology.ResourceTypeArc, segments[len(segments)-1],
	)
	if err != nil {
		return arctext.Text{}, err
	}
	if !ok {
		return arctext.Text{}, notFound
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return arctext.Text{}, err
	}
	var a Arc
	if err = s.NewRetrieve().Where(MatchKeys(key)).Entry(&a).Exec(ctx, tx); err != nil {
		return arctext.Text{}, err
	}
	if a.Type() != TypeModule {
		return arctext.Text{}, errors.Newf(
			"%q is an automation and cannot be imported", path,
		)
	}
	src, ok := a.Base().Source.Variant.(TextSource)
	if !ok {
		return arctext.Text{}, errors.Newf(
			"module %q is a graph and cannot be imported", path,
		)
	}
	return src.Text.Materialize(), nil
}

// childNamed returns the child of parent with the type and name, false if none.
func (s *Service) childNamed(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	typ ontology.ResourceType,
	name string,
) (ontology.ID, bool, error) {
	var children []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(typ).
		Entries(&children).
		Exec(ctx, tx); err != nil {
		return ontology.ID{}, false, err
	}
	child, ok := lo.Find(children, func(r ontology.Resource) bool {
		return r.Name == name
	})
	return child.ID, ok, nil
}
