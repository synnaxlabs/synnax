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

	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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

// modulePaths returns a loader function that lists the import path of each text
// module that readable allows. It reads readable outside the snapshot, since reading
// the access can open a snapshot of its own.
func (s *Service) modulePaths(
	readable func(context.Context, resource.ID) (bool, error),
) func(context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		var (
			dirs []directory.Directory
			arcs []Arc
		)
		if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			if err := s.directories.NewRetrieve().
				Entries(&dirs).
				Exec(ctx, snap); err != nil {
				return err
			}
			return s.NewRetrieve().Entries(&arcs).Exec(ctx, snap)
		}); err != nil {
			return nil, err
		}
		byKey := make(map[directory.Key]directory.Directory, len(dirs))
		for _, d := range dirs {
			byKey[d.Key] = d
		}
		paths := make([]string, 0, len(arcs))
		for _, a := range arcs {
			if _, ok := a.Base().Source.Variant.(TextSource); !ok ||
				a.Type() != TypeModule {
				continue
			}
			ok, err := readable(ctx, a.ResourceID())
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			segments := []string{a.Base().Name}
			for p := a.Base().Directory; p != nil && *p != s.root; p = byKey[*p].Parent {
				segments = append(segments, byKey[*p].Name)
			}
			slices.Reverse(segments)
			paths = append(paths, strings.Join(segments, "."))
		}
		return paths, nil
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
	a, err := s.retrieveModule(ctx, snap, path, filters...)
	if err != nil {
		return arctext.Text{}, err
	}
	src, ok := a.Base().Source.Variant.(TextSource)
	if !ok {
		return arctext.Text{}, errors.Newf(
			"module %q is a graph and cannot be imported", path,
		)
	}
	return src.Text.Materialize(), nil
}

// retrieveModule returns the module at an import path that every filter matches.
func (s *Service) retrieveModule(
	ctx context.Context,
	snap gorp.Snapshot,
	path string,
	filters ...Filter,
) (Arc, error) {
	a, err := s.retrieveArc(ctx, snap, path, filters...)
	if err != nil {
		return Arc{}, err
	}
	if a.Type() != TypeModule {
		return Arc{}, errors.Newf("%q is an automation and cannot be imported", path)
	}
	return a, nil
}

// retrieveArc returns the Arc of either type at an import path that every filter
// matches.
func (s *Service) retrieveArc(
	ctx context.Context,
	snap gorp.Snapshot,
	path string,
	filters ...Filter,
) (Arc, error) {
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
			return Arc{}, err
		}
		if len(dirs) == 0 {
			return Arc{}, notFound
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
		return Arc{}, err
	}
	if len(arcs) == 0 {
		return Arc{}, notFound
	}
	return arcs[0], nil
}
