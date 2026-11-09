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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
)

// moduleSource returns a loader source that reads the text of the module at an
// import path, one directory below Arcs per segment. It sees only the modules that
// every filter matches.
func (s *Service) moduleSource(
	snap gorp.Snapshot,
	filters ...Filter,
) func(context.Context, string) (arctext.Text, error) {
	return func(ctx context.Context, path string) (arctext.Text, error) {
		return s.retrieveModuleText(ctx, snap, path, filters...)
	}
}

// checkedModuleSource returns a loader source like moduleSource that records each read
// in lookups. A read gives the same answer when it gives the same text or error.
func (s *Service) checkedModuleSource(
	snap gorp.Snapshot,
	lookups *channel.Lookups,
) func(context.Context, string) (arctext.Text, error) {
	return func(ctx context.Context, path string) (arctext.Text, error) {
		t, err := s.retrieveModuleText(ctx, snap, path)
		lookups.Record(func(ctx context.Context, snap gorp.Snapshot) (bool, error) {
			again, againErr := s.retrieveModuleText(ctx, snap, path)
			if err != nil || againErr != nil {
				return err != nil && againErr != nil &&
					err.Error() == againErr.Error(), nil
			}
			return t.Raw == again.Raw, nil
		})
		return t, err
	}
}

func (s *Service) retrieveModuleText(
	ctx context.Context,
	snap gorp.Snapshot,
	path string,
	filters ...Filter,
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
			Exec(ctx, snap); err != nil {
			return arctext.Text{}, err
		}
		if len(dirs) == 0 {
			return arctext.Text{}, notFound
		}
		parent = dirs[0].Key
	}
	var arcs []Arc
	q := s.NewRetrieve().Where(And(
		MatchDirectories(parent),
		MatchNames(segments[len(segments)-1]),
	))
	for _, f := range filters {
		q = q.Where(f)
	}
	if err := q.Entries(&arcs).Exec(ctx, snap); err != nil {
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
