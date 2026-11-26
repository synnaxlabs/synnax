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
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// MatchPaths returns a filter for the channels at paths. A path is the names of the
// folders from the top, then the name of the channel, joined by validate.PathSeparator,
// in any normal form. A path that no channel is at, or that passes a folder that
// Retrieve.AllowFolders refuses, matches nothing. Each segment costs one probe of the
// names in a folder, and the filter reads no record to find the keys. It checks the
// folder and the name of each channel it decodes, so a channel that a commit in flight
// renames or moves shows as storage holds it. A folder that a commit in flight renames
// shows under its old name until the commit flushes.
func MatchPaths(paths ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return gorp.MatchKeysChecked[Key, Channel](func(ctx gorp.Context) (
			[]Key,
			func(*Channel) bool,
			error,
		) {
			var (
				keys   = make([]Key, 0, len(paths))
				wanted = make(set.Set[group.Sibling], len(paths))
			)
			for _, p := range paths {
				s, found, err := r.siblingAt(
					ctx,
					ctx.Snapshot,
					validate.NormalizeName(p),
				)
				if err != nil {
					return nil, nil, err
				}
				if !found {
					continue
				}
				wanted.Add(s)
				key, held, err := r.indexes.sibling.Key(ctx, ctx.Snapshot, s)
				if err != nil {
					return nil, nil, err
				}
				if held {
					keys = append(keys, key)
				}
				keys = append(keys, r.indexes.sibling.Committing(s)...)
			}
			return keys, func(ch *Channel) bool { return wanted.Contains(siblingOf(ch)) }, nil
		})
	}
}

// siblingOf returns the folder and the name in NFC of ch.
func siblingOf(ch *Channel) group.Sibling {
	s := group.Sibling{Name: validate.NormalizeName(ch.Name)}
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
		i := strings.IndexByte(path, validate.PathSeparator)
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
			if r.folderAllowed != nil && !r.folderAllowed(id) {
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
// channel, joined by validate.PathSeparator. The texts share one allocation.
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
			b.WriteByte(validate.PathSeparator)
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

// AmbiguousNameError is the error of a name that several channels hold. The Service
// returns it inside a query.MultipleResults error. It names the holders by key and no
// path, since a path can name a folder that the reader cannot retrieve.
// Service.DescribeAmbiguity writes the paths that a subject can see.
type AmbiguousNameError struct {
	// Name is the name, and Holders are the keys of the channels that hold it.
	Name    string
	Holders Keys
}

// Error implements error.
func (e AmbiguousNameError) Error() string {
	keys := make([]string, len(e.Holders))
	for i, k := range e.Holders {
		keys[i] = k.String()
	}
	return fmt.Sprintf(
		"%d channels are named '%s' (keys %s)",
		len(e.Holders),
		e.Name,
		strings.Join(keys, ", "),
	)
}

// quotePath returns path in single quotes, with each quote and backslash escaped.
func quotePath(path string) string {
	var b strings.Builder
	b.Grow(len(path) + 2)
	b.WriteByte('\'')
	for i := range len(path) {
		if path[i] == '\'' || path[i] == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(path[i])
	}
	b.WriteByte('\'')
	return b.String()
}

// AmbiguousName returns the query.MultipleResults error that holds the
// AmbiguousNameError of name and holders.
func AmbiguousName(name string, holders Keys) error {
	err := AmbiguousNameError{Name: name, Holders: holders}
	return query.MultipleResults(err, "%s", err.Error())
}

// KeysByNames returns the key of the channel that holds each of names, compared in NFC,
// as snap holds them. A name that no channel holds adds no key. A name that several
// channels hold returns an AmbiguousNameError.
func (s *Service) KeysByNames(
	ctx context.Context,
	snap gorp.Snapshot,
	names []string,
) (Keys, error) {
	normal := NormalizeNames(names)
	var holders []Channel
	if err := s.newRetrieve().
		Where(MatchNames(normal...)).
		Entries(&holders).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	byName := make(map[string]Keys, len(normal))
	for _, ch := range holders {
		byName[ch.Name] = append(byName[ch.Name], ch.Key())
	}
	for i, n := range normal {
		if keys := byName[n]; len(keys) > 1 {
			return nil, AmbiguousName(names[i], keys)
		}
	}
	return KeysFromChannels(holders), nil
}

// DescribeAmbiguity returns err as a subject sees it when err is an AmbiguousNameError,
// and returns err unchanged otherwise. access returns whether the subject can retrieve
// a record. The result is a query.MultipleResults error that quotes the path of each
// holder that the subject can retrieve along with every folder on its path, then counts
// the other holders. errors.As still finds the AmbiguousNameError in it. Paths are as
// snap holds them.
func (s *Service) DescribeAmbiguity(
	ctx context.Context,
	snap gorp.Snapshot,
	err error,
	access func(context.Context) (func(resource.ID) bool, error),
) error {
	ambiguous, ok := errors.AsType[AmbiguousNameError](err)
	if !ok {
		return err
	}
	allowed, rErr := access(ctx)
	if rErr != nil {
		return rErr
	}
	var holders []Channel
	if rErr = s.newRetrieve().
		Where(MatchKeys(ambiguous.Holders...)).
		Entries(&holders).
		Exec(ctx, snap); rErr != nil {
		return rErr
	}
	paths, rErr := Paths(ctx, snap, holders)
	if rErr != nil {
		return rErr
	}
	folderAllowed := func(g group.Group) bool { return allowed(g.ResourceID()) }
	texts := PathTexts(paths)
	shown := make([]string, 0, len(holders)+1)
	for i, p := range paths {
		if allowed(holders[i].ResourceID()) && lo.EveryBy(p.Folders, folderAllowed) {
			shown = append(shown, quotePath(texts[i]))
		}
	}
	slices.Sort(shown)
	if hidden := len(ambiguous.Holders) - len(shown); hidden > 0 && len(shown) > 0 {
		shown = append(shown, fmt.Sprintf("and %d more you cannot see", hidden))
	} else if hidden > 0 {
		shown = append(shown, fmt.Sprintf("%d you cannot see", hidden))
	}
	return query.MultipleResults(
		err,
		"%d channels are named '%s': %s",
		len(ambiguous.Holders),
		ambiguous.Name,
		strings.Join(shown, ", "),
	)
}
