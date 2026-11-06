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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

// Provider searches the records of one type and retrieves them by ID.
type Provider interface {
	// Search returns the IDs of the records whose text matches q, closest match first.
	// Search reads the changes staged against tx.
	Search(
		ctx context.Context,
		tx gorp.Tx,
		q search.Query[ID],
	) (search.Result[ID], error)
	// Retrieve returns the records that ids name. It leaves out each ID that names no
	// record. Every ID has the type of the Provider.
	Retrieve(ctx context.Context, tx gorp.Tx, ids []ID) ([]Resource, error)
	// List returns at most limit records, in the same order on every call. It leaves
	// out the records whose IDs except returns true for, and still returns limit
	// records when that many remain. A nil except leaves out nothing.
	List(
		ctx context.Context,
		tx gorp.Tx,
		limit int,
		except func(ID) bool,
	) ([]Resource, error)
}

// SearchIndex searches idx as Provider.Search does. id returns the ID of the entry with
// a key.
func SearchIndex[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	idx *gorp.SearchIndex[K, E],
	id func(K) ID,
	q search.Query[ID],
) (search.Result[ID], error) {
	res, err := idx.Search(ctx, tx, search.Query[K]{
		Term:   q.Term,
		Except: func(key K) bool { return q.Except != nil && q.Except(id(key)) },
		Limit:  q.Limit,
	})
	if err != nil {
		return search.Result[ID]{}, err
	}
	return search.MapKeys(res, id), nil
}

// TableProviderConfig is the configuration for a TableProvider.
type TableProviderConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Table stores the records.
	Table *gorp.Table[K, E]
	// Search ranks the records for a search term.
	Search *gorp.SearchIndex[K, E]
	// ID returns the resource ID of the record with a key.
	ID func(K) ID
	// Key parses the key of the record that a resource ID names.
	Key func(ID) (K, error)
	// Name returns the name of a record.
	Name func(*E) string
}

// TableProvider is a Provider for the records of one Gorp table.
type TableProvider[K gorp.Key, E gorp.Entry[K]] struct {
	cfg TableProviderConfig[K, E]
}

// NewTableProvider returns a TableProvider with the given configuration.
func NewTableProvider[K gorp.Key, E gorp.Entry[K]](
	cfg TableProviderConfig[K, E],
) *TableProvider[K, E] {
	return &TableProvider[K, E]{cfg: cfg}
}

// Search implements Provider.
func (p *TableProvider[K, E]) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[ID],
) (search.Result[ID], error) {
	return SearchIndex(ctx, tx, p.cfg.Search, p.cfg.ID, q)
}

// Retrieve implements Provider. It returns the error of the configured Key for an ID
// with a malformed key.
func (p *TableProvider[K, E]) Retrieve(
	ctx context.Context,
	tx gorp.Tx,
	ids []ID,
) ([]Resource, error) {
	keys, err := lo.MapErr(ids, func(id ID, _ int) (K, error) { return p.cfg.Key(id) })
	if err != nil {
		return nil, err
	}
	var entries []E
	if err = p.cfg.Table.NewRetrieve().
		Where(gorp.MatchKeys[K, E](keys...)).
		Entries(&entries).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return p.resources(entries), nil
}

// List implements Provider. It lists the records in the order of their keys.
func (p *TableProvider[K, E]) List(
	ctx context.Context,
	tx gorp.Tx,
	limit int,
	except func(ID) bool,
) ([]Resource, error) {
	q := p.cfg.Table.NewRetrieve().Limit(limit)
	if except != nil {
		q = q.Where(gorp.Match(func(_ gorp.Context, e *E) (bool, error) {
			return !except(p.cfg.ID((*e).GorpKey())), nil
		}))
	}
	var entries []E
	if err := q.Entries(&entries).Exec(ctx, tx); err != nil {
		return nil, err
	}
	return p.resources(entries), nil
}

func (p *TableProvider[K, E]) resources(entries []E) []Resource {
	return lo.Map(entries, func(e E, _ int) Resource {
		return Resource{ID: p.cfg.ID(e.GorpKey()), Name: p.cfg.Name(&e)}
	})
}
