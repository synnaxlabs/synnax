// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

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
	keys := KeysFromResourceIDs(ids)
	var statuses []Status[any]
	if err := s.NewRetrieve[any]().
		Where(MatchKeys[any](keys...)).
		Entries(&statuses).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(statuses, func(e Status[any], _ int) resource.Resource {
		return resource.Resource{ID: e.ResourceID(), Name: e.Name}
	}), nil
}
