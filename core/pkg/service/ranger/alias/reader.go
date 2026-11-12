// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

// maxAncestors is the most parent ranges that a Reader walks up from a range.
const maxAncestors = 64

// ErrAncestorLimit is returned when a Reader walks more than 64 parent ranges up from
// a range without reaching the root.
var ErrAncestorLimit = errors.New("range has too many ancestors")

// ErrAmbiguous is returned when Resolve finds more than one channel that holds an alias
// on a range.
var ErrAmbiguous = errors.Wrap(
	query.ErrUniqueViolation,
	"alias names more than one channel",
)

// Reader is used to retrieve aliases.
type Reader struct {
	snap            gorp.Snapshot
	parentRetriever ParentRetriever
	table           *gorp.Table[string, Alias]
	indexes         indexes
}

// walk calls visit on rng, then on each ancestor of rng, until visit returns true or
// the root range is reached.
func (r Reader) walk(
	ctx context.Context,
	rng ranger.Key,
	visit func(ranger.Key) (bool, error),
) error {
	for range maxAncestors + 1 {
		if done, err := visit(rng); done || err != nil {
			return err
		}
		parent, err := r.parentRetriever.RetrieveParentKey(ctx, rng, r.snap)
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		rng = parent
	}
	return errors.Wrapf(ErrAncestorLimit, "more than %d ancestors", maxAncestors)
}

// Retrieve gets the alias for the given channel on the specified range. If the alias
// is not found on the range, it checks the ancestors of the range.
func (r Reader) Retrieve(
	ctx context.Context,
	rng ranger.Key,
	ch channel.Key,
) (string, error) {
	var res Alias
	found := false
	if err := r.walk(ctx, rng, func(rng ranger.Key) (bool, error) {
		err := r.table.NewRetrieve().
			Where(gorp.MatchKeys[string, Alias](Alias{Range: rng, Channel: ch}.GorpKey())).
			Entry(&res).
			Exec(ctx, r.snap)
		if errors.Is(err, query.ErrNotFound) {
			return false, nil
		}
		found = err == nil
		return found, err
	}); err != nil {
		return "", err
	}
	if !found {
		return "", errors.Wrapf(query.ErrNotFound, "no alias for channel %s", ch)
	}
	return res.Alias, nil
}

// Holders returns the channels that hold the alias on the nearest of the specified
// range and its ancestors that has it. The alias must match exactly. It returns no
// channels when no range has the alias.
func (r Reader) Holders(
	ctx context.Context,
	rng ranger.Key,
	alias string,
) ([]channel.Key, error) {
	var res []Alias
	if err := r.walk(ctx, rng, func(rng ranger.Key) (bool, error) {
		err := r.table.NewRetrieve().
			Where(r.indexes.alias.Filter(alias)).
			Where(gorp.Match(func(_ gorp.Context, a *Alias) (bool, error) {
				return a.Range == rng, nil
			})).
			Entries(&res).
			Exec(ctx, r.snap)
		return len(res) > 0, err
	}); err != nil {
		return nil, err
	}
	return lo.Map(res, func(a Alias, _ int) channel.Key { return a.Channel }), nil
}

// Resolve returns the channel that holds the alias on the specified range. The alias
// must match exactly. If the alias is not found on the range, it checks the ancestors
// of the range. It returns ErrAmbiguous when more than one channel holds the alias on
// the nearest range that has it.
func (r Reader) Resolve(
	ctx context.Context,
	rng ranger.Key,
	alias string,
) (channel.Key, error) {
	holders, err := r.Holders(ctx, rng, alias)
	if err != nil {
		return 0, err
	}
	if len(holders) == 0 {
		return 0, errors.Wrapf(query.ErrNotFound, "no channel with alias %q", alias)
	}
	if len(holders) > 1 {
		return 0, errors.Wrapf(ErrAmbiguous, "alias %q", alias)
	}
	return holders[0], nil
}

// List retrieves all aliases on the specified range, including inherited
// aliases from parent ranges. Local aliases take precedence over inherited
// ones.
func (r Reader) List(
	ctx context.Context,
	rng ranger.Key,
) (map[channel.Key]string, error) {
	res := make(map[channel.Key]string)
	if err := r.walk(ctx, rng, func(rng ranger.Key) (bool, error) {
		var aliases []Alias
		if err := r.table.NewRetrieve().
			WherePrefix([]byte(rng.String()+keySeparator)).
			Entries(&aliases).
			Exec(ctx, r.snap); err != nil {
			return false, err
		}
		for _, a := range aliases {
			if _, ok := res[a.Channel]; !ok {
				res[a.Channel] = a.Alias
			}
		}
		return false, nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// Search returns the channels whose alias on the specified range matches term, closest
// match first. It searches only the aliases of the channels that visible accepts, and
// not the aliases of parent ranges.
func (r Reader) Search(
	ctx context.Context,
	rng ranger.Key,
	term string,
	visible func(channel.Key) bool,
) ([]channel.Key, error) {
	var aliases []Alias
	if err := r.table.NewRetrieve().
		WherePrefix([]byte(rng.String()+keySeparator)).
		Entries(&aliases).
		Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	idx := search.New[channel.Key]()
	for _, a := range aliases {
		if visible(a.Channel) {
			idx.Set(a.Channel, a.Alias)
		}
	}
	return idx.Search(search.Query[channel.Key]{Term: term}).Keys, nil
}

// SearchChannels returns the channels that q selects and that match term by name or by
// their alias on rng. Alias matches come first, then name matches, each closest match
// first. It matches only the aliases of the channels that visible accepts. A positive
// limit caps the merged list.
func (r Reader) SearchChannels(
	ctx context.Context,
	q channel.Retrieve,
	rng ranger.Key,
	term string,
	visible func(channel.Key) bool,
	limit int,
) ([]channel.Channel, error) {
	keys, err := r.Search(ctx, rng, term, visible)
	if err != nil {
		return nil, err
	}
	var aliased []channel.Channel
	if len(keys) > 0 {
		// An alias can outlive its channel.
		if err := q.SkipMissingKeys().
			Where(channel.MatchKeys(keys...)).
			Entries(&aliased).
			Exec(ctx, r.snap); err != nil {
			return nil, err
		}
	}
	matched := lo.KeyBy(aliased, func(ch channel.Channel) channel.Key {
		return ch.Key()
	})
	named := q.Search(term)
	if limit > 0 {
		named = named.Limit(limit)
	}
	var byName []channel.Channel
	if err := named.Entries(&byName).Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	merged := lo.FilterMap(keys, func(k channel.Key, _ int) (channel.Channel, bool) {
		ch, ok := matched[k]
		return ch, ok
	})
	merged = append(merged, lo.Reject(byName, func(ch channel.Channel, _ int) bool {
		_, ok := matched[ch.Key()]
		return ok
	})...)
	if limit > 0 {
		merged = merged[:min(limit, len(merged))]
	}
	return merged, nil
}
