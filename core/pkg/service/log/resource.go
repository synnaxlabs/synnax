// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"
	"uuid"

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
		return uuid.Parse(id.Key)
	})
	if err != nil {
		return nil, err
	}
	var logs []Log
	if err = s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&logs).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(logs, func(e Log, _ int) resource.Resource {
		return resource.Resource{ID: e.ResourceID(), Name: e.Name}
	}), nil
}
