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
	"regexp"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

// Reader is used to retrieve aliases.
type Reader struct {
	tx              gorp.Tx
	parentRetriever ParentRetriever
	table           *gorp.Table[string, Alias]
}

// Retrieve gets the alias for the given channel on the specified range.
// If the alias is not found on the range, it will recursively check parent
// ranges.
func (r Reader) Retrieve(
	ctx context.Context,
	rng ranger.Key,
	ch channel.Key,
) (string, error) {
	var res Alias
	err := r.table.NewRetrieve().
		Where(gorp.MatchKeys[string, Alias](Alias{Range: rng, Channel: ch}.GorpKey())).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.ErrNotFound) {
		if r.parentRetriever == nil {
			return res.Alias, err
		}
		parentKey, pErr := r.parentRetriever.RetrieveParentKey(ctx, rng, r.tx)
		if errors.Is(pErr, query.ErrNotFound) {
			return res.Alias, err
		}
		if pErr != nil {
			return "", pErr
		}
		return r.Retrieve(ctx, parentKey, ch)
	}
	return res.Alias, err
}

// Resolve attempts to resolve the given alias to a channel key on the
// specified range. Supports regex matching. If not found on the range, it will
// recursively check parent ranges.
func (r Reader) Resolve(
	ctx context.Context,
	rng ranger.Key,
	alias string,
) (channel.Key, error) {
	var res Alias
	matcher := func(ctx gorp.Context, a *Alias) (bool, error) {
		return a.Range == rng && a.Alias == alias, nil
	}
	rxp, err := regexp.Compile(alias)
	if err == nil {
		matcher = func(ctx gorp.Context, a *Alias) (bool, error) {
			return a.Range == rng && rxp.MatchString(a.Alias), nil
		}
	}
	err = r.table.NewRetrieve().
		Where(gorp.Match(matcher)).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.ErrNotFound) {
		if r.parentRetriever == nil {
			return 0, err
		}
		parentKey, pErr := r.parentRetriever.RetrieveParentKey(ctx, rng, r.tx)
		if errors.Is(pErr, query.ErrNotFound) {
			return 0, err
		}
		if pErr != nil {
			return 0, pErr
		}
		return r.Resolve(ctx, parentKey, alias)
	}
	if err != nil {
		return 0, err
	}
	return res.Channel, nil
}

// List retrieves all aliases on the specified range, including inherited
// aliases from parent ranges. Local aliases take precedence over inherited
// ones.
func (r Reader) List(
	ctx context.Context,
	rng ranger.Key,
) (map[channel.Key]string, error) {
	res := make(map[channel.Key]string)
	if err := r.listAliases(ctx, rng, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (r Reader) listAliases(
	ctx context.Context,
	rng ranger.Key,
	accumulated map[channel.Key]string,
) error {
	var aliases []Alias
	if err := r.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, a *Alias) (bool, error) {
			return a.Range == rng, nil
		})).
		Entries(&aliases).
		Exec(ctx, r.tx); err != nil {
		return err
	}
	for _, a := range aliases {
		accumulated[a.Channel] = a.Alias
	}
	if r.parentRetriever == nil {
		return nil
	}
	parentKey, pErr := r.parentRetriever.RetrieveParentKey(ctx, rng, r.tx)
	if errors.Is(pErr, query.ErrNotFound) {
		return nil
	} else if pErr != nil {
		return pErr
	}
	return r.listAliases(ctx, parentKey, accumulated)
}

// Search returns the channels whose alias on the specified range matches term, closest
// match first. It does not search the aliases of parent ranges.
func (r Reader) Search(
	ctx context.Context,
	rng ranger.Key,
	term string,
) ([]channel.Key, error) {
	var aliases []Alias
	if err := r.table.NewRetrieve().
		WherePrefix([]byte(rng.String()+keySeparator)).
		Entries(&aliases).
		Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	idx := search.New[channel.Key]()
	for _, a := range aliases {
		idx.Set(a.Channel, a.Alias)
	}
	return idx.Search(term, 0).Keys, nil
}

// SearchChannels returns the channels that q selects and that match term by name or by
// their alias on rng. Alias matches come first, then name matches, each closest match
// first. A positive limit and the offset page the merged list.
func (r Reader) SearchChannels(
	ctx context.Context,
	q channel.Retrieve,
	rng ranger.Key,
	term string,
	limit, offset int,
) ([]channel.Channel, error) {
	keys, err := r.Search(ctx, rng, term)
	if err != nil {
		return nil, err
	}
	var aliased []channel.Channel
	if len(keys) > 0 {
		// An alias can outlive its channel.
		if err := q.SkipMissingKeys().
			Where(channel.MatchKeys(keys...)).
			Entries(&aliased).
			Exec(ctx, r.tx); err != nil {
			return nil, err
		}
	}
	matched := lo.KeyBy(aliased, func(ch channel.Channel) channel.Key {
		return ch.Key()
	})
	named := q.Search(term)
	if limit > 0 {
		named = named.Limit(limit + offset)
	}
	var byName []channel.Channel
	if err := named.Entries(&byName).Exec(ctx, r.tx); err != nil {
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
	length := uint(len(merged))
	if limit > 0 {
		length = uint(limit)
	}
	return lo.Subset(merged, offset, length), nil
}
