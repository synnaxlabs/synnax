// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// MatchPaths returns a filter for the channels at paths. A path is the names of the
// folders from the top, then the name of the channel, joined by '.', in any normal
// form. A path that no channel is at, or that passes a folder that
// Retrieve.AllowFolders refuses, matches nothing. Each folder and channel of a path
// matches as storage holds it, also while a commit is in flight: each segment reads the
// folder that holds its name, and the filter checks the folder and the name of each
// channel that it decodes.
func MatchPaths(paths ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return r.indexes.sibling.FilterFunc(func(ctx gorp.Context) (
			[]group.Sibling,
			error,
		) {
			siblings := make([]group.Sibling, 0, len(paths))
			for _, p := range paths {
				s, found, err := r.siblingAt(
					ctx,
					ctx.Snapshot,
					normalizeName(p),
				)
				if err != nil {
					return nil, err
				}
				if found {
					siblings = append(siblings, s)
				}
			}
			return siblings, nil
		})
	}
}

// siblingOf returns the folder and the name of ch.
func siblingOf(ch *Channel) group.Sibling {
	s := group.Sibling{Name: ch.Name}
	if ch.Group != nil {
		s.Parent = *ch.Group
	}
	return s
}

// siblingAt returns the folder of the last segment of path and that segment, as seen
// by snap. It returns false when a folder on the path does not exist or r does not
// allow it.
func (r Retrieve) siblingAt(ctx context.Context, snap gorp.Snapshot, path string) (
	group.Sibling,
	bool,
	error,
) {
	var parent group.Key
	for {
		i := strings.IndexByte(path, pathSeparator)
		if i < 0 {
			return group.Sibling{Parent: parent, Name: path}, true, nil
		}
		folder := group.Sibling{Parent: parent, Name: path[:i]}
		holder, found, err := r.indexes.sibling.Holder(ctx, snap, folder)
		if err != nil || !found {
			return group.Sibling{}, false, err
		}
		switch holder.Table {
		case tableName:
			return group.Sibling{}, false, nil
		case string(resource.TypeGroup):
			id := resource.ID{Type: resource.TypeGroup, Key: holder.Key}
			if parent, err = group.KeyFromResourceID(id); err != nil {
				return group.Sibling{}, false, err
			}
			if allowed := r.folders.allowed; allowed != nil && !allowed(id) {
				return group.Sibling{}, false, nil
			}
		default:
			return group.Sibling{}, false, errors.Newf(
				"%s %s holds a name among channels",
				holder.Table,
				holder.Key,
			)
		}
		path = path[i+1:]
	}
}

// Path is where a channel sits: the folders from the top down to its folder, then its
// name.
type Path struct {
	Folders group.Path
	Name    string
}

// PathTexts returns the text of each of paths: the names of its folders and of its
// channel, joined by '.'. The texts share one allocation.
func PathTexts(paths []Path) []string {
	n := 0
	for _, p := range paths {
		n += len(p.Name) + len(p.Folders)
		for _, g := range p.Folders {
			n += len(g.Name)
		}
	}
	var b strings.Builder
	b.Grow(n)
	ends := make([]int, len(paths))
	for i, p := range paths {
		for _, g := range p.Folders {
			b.WriteString(g.Name)
			b.WriteByte(pathSeparator)
		}
		b.WriteString(p.Name)
		ends[i] = b.Len()
	}
	all, start := b.String(), 0
	texts := make([]string, len(paths))
	for i, end := range ends {
		texts[i], start = all[start:end], end
	}
	return texts
}

// Paths returns the path of each of channels, as seen by snap. Channels in one folder
// share one group.Path.
func Paths(
	ctx context.Context,
	snap gorp.Snapshot,
	channels []Channel,
) ([]Path, error) {
	var keys []group.Key
	for _, ch := range channels {
		if ch.Group != nil {
			keys = append(keys, *ch.Group)
		}
	}
	folders, err := group.Paths(ctx, snap, lo.Uniq(keys)...)
	if err != nil {
		return nil, err
	}
	paths := make([]Path, len(channels))
	for i, ch := range channels {
		paths[i].Name = ch.Name
		if ch.Group != nil {
			paths[i].Folders = folders[*ch.Group]
		}
	}
	return paths, nil
}

// visible reports whether a subject that allowed accepts can see each folder of p.
func (p Path) visible(allowed func(resource.ID) bool) bool {
	return lo.EveryBy(p.Folders, func(g group.Group) bool {
		return allowed(g.ResourceID())
	})
}

// KeysByPaths returns the key of the channel at each of paths, as a subject that
// allowed accepts sees snap, or as anyone sees it when allowed is nil. A path that
// holds only digits is a key. Any other path is the names of the folders from the top
// level down and the name of the channel, joined by '.', so a path with no '.' is a
// channel at the top level. Names compare in NFC. A channel that allowed refuses is
// missing, and so is a path through a folder that it refuses. A key finds a channel in
// any folder. The first missing path returns a query.ErrNotFound error that names the
// paths of the channels of its last name that the subject can see.
func (s *Service) KeysByPaths(
	ctx context.Context,
	snap gorp.Snapshot,
	allowed func(resource.ID) bool,
	paths []string,
) (Keys, error) {
	if allowed == nil {
		allowed = func(resource.ID) bool { return true }
	}
	keys := make(Keys, 0, len(paths))
	for _, path := range paths {
		match := MatchPaths(path)
		if key, err := strconv.ParseUint(path, 10, 32); err == nil {
			match = MatchKeys(Key(key))
		}
		var found []Channel
		if err := s.NewRetrieve().
			AllowFolders(allowed).
			Where(match).
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		found = lo.Filter(found, func(ch Channel, _ int) bool {
			return allowed(ch.ResourceID())
		})
		if len(found) == 0 {
			return nil, s.notFound(ctx, snap, allowed, path)
		}
		keys = append(keys, found[0].Key())
	}
	return keys, nil
}

// notFound returns the query.ErrNotFound error of path, which names the path of each
// channel of its last name that a subject that allowed accepts can see.
func (s *Service) notFound(
	ctx context.Context,
	snap gorp.Snapshot,
	allowed func(resource.ID) bool,
	path string,
) error {
	name := path[strings.LastIndexByte(path, pathSeparator)+1:]
	var named []Channel
	if err := s.NewRetrieve().
		Where(MatchNames(normalizeName(name))).
		Entries(&named).
		Exec(ctx, snap); err != nil {
		return err
	}
	named = lo.Filter(named, func(ch Channel, _ int) bool {
		return allowed(ch.ResourceID())
	})
	paths, err := Paths(ctx, snap, named)
	if err != nil {
		return err
	}
	texts := PathTexts(paths)
	var seen []string
	for i, p := range paths {
		if p.visible(allowed) {
			seen = append(seen, texts[i])
		}
	}
	if len(seen) == 0 {
		return query.NotFound("%s does not exist", path)
	}
	slices.Sort(seen)
	return query.NotFound(
		"%s does not exist. Did you mean %s?",
		path,
		strings.Join(seen, ", "),
	)
}
