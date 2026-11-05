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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

var _ resource.Provider = (*Service)(nil)

// resourceID returns the resource ID of the alias with the given gorp key.
func resourceID(key string) resource.ID {
	return resource.ID{Type: resource.TypeRangeAlias, Key: key}
}

// Search implements resource.Provider.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, s.search, resourceID, q)
}

// RetrieveResources implements resource.Provider.
func (s *Service) RetrieveResources(
	ctx context.Context,
	tx gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	keys, err := lo.MapErr(ids, func(id resource.ID, _ int) (string, error) {
		r, ch, err := parseGorpKey(id.Key)
		return Alias{Range: r, Channel: ch}.GorpKey(), err
	})
	if err != nil {
		return nil, err
	}
	var aliases []Alias
	if err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, Alias](keys...)).
		Entries(&aliases).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(aliases, toResource), nil
}

// ListResources implements resource.Provider.
func (s *Service) ListResources(
	ctx context.Context,
	tx gorp.Tx,
	limit int,
) ([]resource.Resource, error) {
	var aliases []Alias
	if err := s.table.NewRetrieve().
		Limit(limit).
		Entries(&aliases).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(aliases, toResource), nil
}

func toResource(a Alias, _ int) resource.Resource {
	return resource.Resource{ID: ResourceID(a.Range, a.Channel), Name: a.Alias}
}
