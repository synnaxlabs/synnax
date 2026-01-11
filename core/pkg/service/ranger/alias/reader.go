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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/uuid"
)

// Reader is used to retrieve aliases.
type Reader struct {
	tx              gorp.Tx
	otg             *ontology.Ontology
	parentRetriever ParentRetriever
}

// Retrieve gets the alias for the given channel on the specified range.
// If the alias is not found on the range, it will recursively check parent ranges.
func (r Reader) Retrieve(ctx context.Context, rng uuid.UUID, ch channel.Key) (string, error) {
	var res Alias
	err := gorp.NewRetrieve[string, Alias]().
		WhereKeys(Alias{Range: rng, Channel: ch}.GorpKey()).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.NotFound) {
		if r.parentRetriever == nil {
			return res.Alias, err
		}
		parentKey, pErr := r.parentRetriever.RetrieveParentKey(ctx, rng, r.tx)
		if errors.Is(pErr, query.NotFound) {
			return res.Alias, err
		}
		if pErr != nil {
			return "", pErr
		}
		return r.Retrieve(ctx, parentKey, ch)
	}
	return res.Alias, err
}

// Resolve attempts to resolve the given alias to a channel key on the specified range.
// Supports regex matching. If not found on the range, it will recursively check parent ranges.
func (r Reader) Resolve(ctx context.Context, rng uuid.UUID, alias string) (channel.Key, error) {
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
	err = gorp.
		NewRetrieve[string, Alias]().
		Where(matcher).
		Entry(&res).
		Exec(ctx, r.tx)
	if errors.Is(err, query.NotFound) {
		if r.parentRetriever == nil {
			return 0, err
		}
		parentKey, pErr := r.parentRetriever.RetrieveParentKey(ctx, rng, r.tx)
		if errors.Is(pErr, query.NotFound) {
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

// List retrieves all aliases on the specified range, including inherited aliases from
// parent ranges. Local aliases take precedence over inherited ones.
func (r Reader) List(ctx context.Context, rng uuid.UUID) (map[channel.Key]string, error) {
	res := make(map[channel.Key]string)
	if err := r.listAliases(ctx, rng, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (r Reader) listAliases(
	ctx context.Context,
	rng uuid.UUID,
	accumulated map[channel.Key]string,
) error {
	var aliases []Alias
	if err := gorp.NewRetrieve[string, Alias]().
		Where(func(_ gorp.Context, a *Alias) (bool, error) {
			return a.Range == rng, nil
		}).
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
	if errors.Is(pErr, query.NotFound) {
		return nil
	} else if pErr != nil {
		return pErr
	}
	return r.listAliases(ctx, parentKey, accumulated)
}

// Search searches for aliases fuzzily matching the given term on the specified range.
func (r Reader) Search(ctx context.Context, rng uuid.UUID, term string) ([]channel.Key, error) {
	ids, err := r.otg.SearchIDs(
		ctx,
		ontology.SearchRequest{Term: term, Type: OntologyType},
	)
	if err != nil {
		return nil, err
	}
	res := make([]channel.Key, 0)
	for _, id := range ids {
		rangeKey, chKey, err := parseGorpKey(id.Key)
		if err != nil {
			return nil, err
		}
		if rangeKey == rng {
			res = append(res, chKey)
		}
	}
	return res, nil
}
