// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Resource is the ID and name of a record of any type.
type Resource struct {
	// ID identifies the record.
	ID ID `json:"id" msgpack:"id"`
	// Name is the human-readable name of the record.
	Name string `json:"name" msgpack:"name"`
}

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB is the database that a nil transaction reads from.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Providers searches and retrieves the records of each type.
	//
	// [REQUIRED]
	Providers map[Type]Provider
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Providers = override.Nil(c.Providers, other.Providers)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("resource")
	v.NotNil("db", c.DB)
	v.NotNil("providers", c.Providers)
	for t, p := range c.Providers {
		v.NotNil(fmt.Sprintf("providers.%s", t), p)
	}
	return v.Error()
}

// Service searches and retrieves records of every type that it has a Provider for.
type Service struct {
	cfg ServiceConfig
	// types holds the type of each Provider in sorted order, so Search ranks equal
	// matches the same way on every call.
	types []Type
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, types: slices.Sorted(maps.Keys(cfg.Providers))}, nil
}

const (
	// defaultSearchLimit is the number of resources that Search returns when a
	// SearchQuery sets no Limit.
	defaultSearchLimit = 100
	maxSearchLimit     = 1000
	maxSearchOffset    = 10000
)

// SearchQuery selects the resources that Service.Search returns.
type SearchQuery struct {
	// Term is the text that a resource must match. An empty term matches every
	// resource.
	Term string
	// Types limits the search to resources of these types. Empty searches every type.
	Types []Type
	// Limit is the maximum number of resources to return, at most 1000. Zero returns
	// at most 100.
	Limit int
	// Offset is the number of closest matches to skip, at most 10000.
	Offset int
	// Except, when non-nil, leaves out the resources whose IDs it returns true for.
	// Search still fills each page with the resources that remain.
	Except func(ID) bool
}

// Search returns the resources that match q, closest match first. For an empty term,
// it lists the resources in order of type, then in the order of each Provider. Search
// reads the changes staged against tx, falling back to the database when tx is nil. A
// limit or offset out of bounds, or a type with no Provider, is a validation error.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q SearchQuery,
) ([]Resource, error) {
	v := validate.New("resource.search")
	v.GreaterThanEq("limit", q.Limit, 0)
	v.LessThanEq("limit", q.Limit, maxSearchLimit)
	v.GreaterThanEq("offset", q.Offset, 0)
	v.LessThanEq("offset", q.Offset, maxSearchOffset)
	for _, t := range q.Types {
		s.validateType(v, "types", t)
	}
	if err := v.Error(); err != nil {
		return nil, err
	}
	tx = s.cfg.DB.OverrideTx(tx)
	limit := q.Limit
	if limit == 0 {
		limit = defaultSearchLimit
	}
	if q.Term == "" {
		return s.list(ctx, tx, q, limit)
	}
	results := make([]search.Result[ID], 0, len(s.types))
	for _, t := range s.types {
		if len(q.Types) > 0 && !slices.Contains(q.Types, t) {
			continue
		}
		res, err := s.cfg.Providers[t].Search(ctx, tx, search.Query[ID]{
			Term:   q.Term,
			Except: q.Except,
			Limit:  q.Offset + limit,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	ids := search.Merge(q.Offset+limit, results...).Keys
	return s.retrieve(ctx, tx, ids[min(q.Offset, len(ids)):])
}

func (s *Service) list(
	ctx context.Context,
	tx gorp.Tx,
	q SearchQuery,
	limit int,
) ([]Resource, error) {
	resources := make([]Resource, 0, limit)
	for _, t := range s.types {
		if len(resources) >= q.Offset+limit {
			break
		}
		if len(q.Types) > 0 && !slices.Contains(q.Types, t) {
			continue
		}
		res, err := s.cfg.Providers[t].List(
			ctx,
			tx,
			q.Offset+limit-len(resources),
			q.Except,
		)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res...)
	}
	return resources[min(q.Offset, len(resources)):], nil
}

// Retrieve returns the resources that ids name, in the order of ids. It leaves out each
// ID that names no resource, and returns each resource once. Retrieve reads the changes
// staged against tx, falling back to the database when tx is nil. An ID of a type with
// no Provider is a validation error.
func (s *Service) Retrieve(
	ctx context.Context,
	tx gorp.Tx,
	ids ...ID,
) ([]Resource, error) {
	v := validate.New("resource.retrieve")
	for _, id := range ids {
		s.validateType(v, "ids", id.Type)
	}
	if err := v.Error(); err != nil {
		return nil, err
	}
	return s.retrieve(ctx, s.cfg.DB.OverrideTx(tx), ids)
}

func (s *Service) validateType(v *validate.Validator, field string, t Type) {
	_, ok := s.cfg.Providers[t]
	v.Ternaryf(field, !ok, "unknown resource type %q", t)
}

// retrieve panics when an ID has a type with no Provider, because its callers pass only
// IDs whose types they have validated or that a Provider returned.
func (s *Service) retrieve(
	ctx context.Context,
	tx gorp.Tx,
	ids []ID,
) ([]Resource, error) {
	byType := make(map[Type][]ID)
	seen := make(set.Set[ID], len(ids))
	for _, id := range ids {
		if seen.Contains(id) {
			continue
		}
		seen.Add(id)
		byType[id.Type] = append(byType[id.Type], id)
	}
	found := make(map[ID]Resource, len(seen))
	for t, typeIDs := range byType {
		p, ok := s.cfg.Providers[t]
		if !ok {
			panic(fmt.Sprintf("[resource] - no provider for type %s", t))
		}
		res, err := p.Retrieve(ctx, tx, typeIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			found[r.ID] = r
		}
	}
	resources := make([]Resource, 0, len(found))
	for _, id := range ids {
		if r, ok := found[id]; ok {
			resources = append(resources, r)
			delete(found, id)
		}
	}
	return resources, nil
}
