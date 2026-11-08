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

	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
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
	parent := s.root
	for _, name := range segments[:len(segments)-1] {
		var dirs []directory.Directory
		if err := s.directories.NewRetrieve().
			Where(directory.And(
				directory.MatchParents(parent),
				directory.MatchNames(name),
			)).
			Entries(&dirs).
			Exec(ctx, tx); err != nil {
			return arctext.Text{}, err
		}
		if len(dirs) == 0 {
			return arctext.Text{}, notFound
		}
		parent = dirs[0].Key
	}
	var arcs []Arc
	if err := s.NewRetrieve().
		Where(And(
			MatchDirectories(parent),
			MatchNames(segments[len(segments)-1]),
		)).
		Entries(&arcs).
		Exec(ctx, tx); err != nil {
		return arctext.Text{}, err
	}
	if len(arcs) == 0 {
		return arctext.Text{}, notFound
	}
	a := arcs[0]
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
