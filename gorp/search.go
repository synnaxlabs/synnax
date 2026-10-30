// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/keyset"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/search"
)

// SearchIndex is an in-memory full-text index on text extracted from entries of type
// E. It matches each word of a search term against the words of that text exactly, by
// prefix, or within one edit. Construct via NewSearchIndex and register on a Table
// through TableConfig.Indexes.
type SearchIndex[K Key, E Entry[K]] struct {
	// name is the human-readable index name, used in diagnostics.
	name string
	// text reads the indexed text from an entry.
	text  func(e *E) []string
	store *index.Search[*txState, K]
}

// NewSearchIndex constructs a SearchIndex over the text that text returns for each
// entry. The returned index is empty; register it on a Table through
// TableConfig.Indexes to populate it from the existing table contents and keep it in
// sync with future writes.
func NewSearchIndex[K Key, E Entry[K]](
	name string,
	text func(e *E) []string,
) *SearchIndex[K, E] {
	return &SearchIndex[K, E]{
		name:  name,
		text:  text,
		store: index.NewSearch[*txState, K](),
	}
}

// Name implements Index.
func (s *SearchIndex[K, E]) Name() string { return s.name }

//nolint:unused
func (s *SearchIndex[K, E]) populate() (func(E), func(error)) {
	insert, finish := s.store.Populate()
	return func(entry E) { insert(entry.GorpKey(), s.text(&entry)) },
		func(err error) { finish(populateErr(s.name, err)) }
}

//nolint:unused
func (s *SearchIndex[K, E]) set(entry E) {
	s.store.Set(entry.GorpKey(), s.text(&entry))
}

//nolint:unused
func (s *SearchIndex[K, E]) delete(key K) { s.store.Delete(key) }

//nolint:unused
func (s *SearchIndex[K, E]) stageSet(scope *txState, entry E) {
	s.store.Stage(scope, entry.GorpKey(), s.text(&entry))
}

//nolint:unused
func (s *SearchIndex[K, E]) stageDelete(scope *txState, key K) {
	s.store.StageDelete(scope, key)
}

// Search returns the keys of the entries that match term, closest match first. When no
// entry matches every word of term, it returns the entries that match any word. It
// returns at most limit keys, or every key when limit is 0. Search merges committed
// index state with the changes staged against tx; pass the DB to read committed state
// only. Search panics if tx is nil.
//
// Search blocks until the index has finished populating. If population failed, Search
// scans every entry through tx instead, so it returns the same result at scan cost.
func (s *SearchIndex[K, E]) Search(
	ctx context.Context,
	tx Tx,
	term string,
	limit int,
) (search.Result[K], error) {
	checkForNilTx("SearchIndex.Search", tx)
	if err := s.store.Wait(ctx); errors.Is(err, ErrIndexInvalid) {
		return s.scan(ctx, tx, term, limit)
	} else if err != nil {
		return search.Result[K]{}, err
	}
	return s.store.Get(scopeOf(tx), term, limit)
}

// Filter returns a Filter matching the entries that Search returns for term. A
// Retrieve returns the entries closest match first, also when other filters narrow
// them, so Limit and Offset page through them by relevance.
func (s *SearchIndex[K, E]) Filter(term string) Filter[K, E] {
	return Filter[K, E]{
		resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			res, err := s.Search(ctx, tx, term, 0)
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{keys: keyset.Ranked(res.Keys)}, nil
		},
	}
}

// scan searches every entry that tx reads.
func (s *SearchIndex[K, E]) scan(
	ctx context.Context,
	tx Tx,
	term string,
	limit int,
) (search.Result[K], error) {
	entries, closer, err := wrapReader[K, E](tx, nil).OpenNexter(ctx)
	if err != nil {
		return search.Result[K]{}, err
	}
	idx := search.New[K]()
	for e := range entries {
		idx.Set(e.GorpKey(), s.text(&e)...)
	}
	if err = closer.Close(); err != nil {
		return search.Result[K]{}, err
	}
	return idx.Search(term, limit), nil
}
