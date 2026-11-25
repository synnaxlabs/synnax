// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

// ResourceID returns the resource ID of the calculation with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeCalculation, Key: k.String()}
}

// Resources returns the resource.Provider of calculations. A calculation takes the name
// of its output channel.
func (s *Service) Resources() resource.Provider { return provider{svc: s} }

type provider struct{ svc *Service }

// Search implements resource.Provider. A search finds a calculation by its output
// channel, so it finds none here.
func (provider) Search(
	context.Context,
	gorp.Snapshot,
	search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return search.Result[resource.ID]{}, nil
}

// Retrieve implements resource.Provider.
func (p provider) Retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	ids []resource.ID,
) ([]resource.Resource, error) {
	keys, err := lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
	if err != nil {
		return nil, err
	}
	var calcs []Calculation
	if err = p.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Calculation](keys...)).
		Entries(&calcs).
		Exec(ctx, snap); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return p.resources(ctx, snap, calcs)
}

// List implements resource.Provider. It lists the calculations in the order of their
// keys.
func (p provider) List(
	ctx context.Context,
	snap gorp.Snapshot,
	limit int,
	except func(resource.ID) bool,
) ([]resource.Resource, error) {
	q := p.svc.table.NewRetrieve().Limit(limit)
	if except != nil {
		q = q.Where(gorp.Match(func(_ gorp.Context, c *Calculation) (bool, error) {
			return !except(ResourceID(c.Key)), nil
		}))
	}
	var calcs []Calculation
	if err := q.Entries(&calcs).Exec(ctx, snap); err != nil {
		return nil, err
	}
	return p.resources(ctx, snap, calcs)
}

func (p provider) resources(
	ctx context.Context,
	snap gorp.Snapshot,
	calcs []Calculation,
) ([]resource.Resource, error) {
	outputs := lo.Map(calcs, func(c Calculation, _ int) channel.Key { return c.Output })
	var channels []channel.Channel
	if err := p.svc.channels.NewRetrieve().
		Where(channel.MatchKeys(outputs...)).
		SkipMissingKeys().
		Entries(&channels).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	names := make(map[channel.Key]string, len(channels))
	for _, ch := range channels {
		names[ch.Key()] = ch.Name
	}
	return lo.Map(calcs, func(c Calculation, _ int) resource.Resource {
		return resource.Resource{ID: ResourceID(c.Key), Name: names[c.Output]}
	}), nil
}
