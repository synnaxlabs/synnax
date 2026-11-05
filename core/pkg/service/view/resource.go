// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

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
	return resource.SearchIndex(ctx, tx, s.search, ResourceID, q)
}

// RetrieveResources implements resource.Provider.
func (s *Service) RetrieveResources(
	ctx context.Context,
	tx gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	keys, err := KeysFromResourceIDs(ids)
	if err != nil {
		return nil, err
	}
	var views []View
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&views).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(views, toResource), nil
}

// ListResources implements resource.Provider.
func (s *Service) ListResources(
	ctx context.Context,
	tx gorp.Tx,
	limit int,
) ([]resource.Resource, error) {
	var views []View
	if err := s.NewRetrieve().Limit(limit).Entries(&views).Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(views, toResource), nil
}

func toResource(e View, _ int) resource.Resource {
	return resource.Resource{ID: e.ResourceID(), Name: e.Base().Name}
}
