// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

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

// Search implements resource.Provider.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, s.indexes.search, ResourceID, q)
}

// RetrieveResources implements resource.Provider.
func (s *Service) RetrieveResources(
	ctx context.Context,
	tx gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	keys, err := lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
	if err != nil {
		return nil, err
	}
	var groups []Group
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&groups).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(groups, toResource), nil
}

// ListResources implements resource.Provider.
func (s *Service) ListResources(
	ctx context.Context,
	tx gorp.Tx,
	limit int,
) ([]resource.Resource, error) {
	var groups []Group
	if err := s.NewRetrieve().Limit(limit).Entries(&groups).Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(groups, toResource), nil
}

func toResource(e Group, _ int) resource.Resource {
	return resource.Resource{ID: e.ResourceID(), Name: e.Name}
}
