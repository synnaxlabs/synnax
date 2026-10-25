// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"
	"sync"

	"github.com/synnaxlabs/x/search"
)

// Search is a full-text index of the text that each primary key of type K holds. Search
// is safe for concurrent use.
type Search[S Scope, K comparable] struct {
	// mu guards population.err, and orders writes after the bulk load. Populate holds
	// it for writing until the load finishes.
	mu         sync.RWMutex
	population population
	committed  *search.Index[K]
	overlay    overlay[S, texts[K]]
}

// texts is the text that one scope has staged for each key. A nil text marks a deleted
// key.
type texts[K comparable] map[K][]string

func (t texts[K]) isEmpty() bool { return len(t) == 0 }

// NewSearch returns an empty Search that waits for its bulk load.
func NewSearch[S Scope, K comparable]() *Search[S, K] {
	s := &Search[S, K]{population: newPopulation(), committed: search.New[K]()}
	s.overlay.create = func() texts[K] { return make(texts[K]) }
	s.overlay.flush = s.flush
	return s
}

// Populate starts the bulk load and locks the index until it finishes. insert adds the
// committed text of a key. finish ends the load and must be called exactly once; a
// non-nil err makes the index invalid, and every later read returns err.
func (s *Search[S, K]) Populate() (insert func(K, []string), finish func(err error)) {
	s.mu.Lock()
	return func(key K, text []string) { s.committed.Set(key, text...) },
		func(err error) {
			s.population.finish(err)
			s.mu.Unlock()
		}
}

// Wait blocks until the bulk load finishes or ctx is done. It returns the load's error
// or ctx.Err().
func (s *Search[S, K]) Wait(ctx context.Context) error {
	return s.population.wait(ctx)
}

// Set replaces the committed text of key.
func (s *Search[S, K]) Set(key K, text []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committed.Set(key, text...)
}

// Delete removes key from committed state.
func (s *Search[S, K]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committed.Delete(key)
}

// Stage replaces the text of key in the changes of scope. The change applies to
// committed state when scope commits.
func (s *Search[S, K]) Stage(scope S, key K, text []string) {
	if isNone(scope) {
		s.Set(key, text)
		return
	}
	if text == nil {
		text = []string{}
	}
	s.overlay.load(scope)[key] = text
}

// StageDelete removes key in the changes of scope. The removal applies to committed
// state when scope commits.
func (s *Search[S, K]) StageDelete(scope S, key K) {
	if isNone(scope) {
		s.Delete(key)
		return
	}
	s.overlay.load(scope)[key] = nil
}

// Get returns the keys whose text matches term, closest match first, with the changes
// of scope applied. It blocks until the bulk load finishes.
func (s *Search[S, K]) Get(scope S, term string) (search.Result[K], error) {
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.population.err; err != nil {
		return search.Result[K]{}, err
	}
	staged, ok := s.overlay.get(scope)
	if !ok {
		return s.committed.Search(term), nil
	}
	pending := search.New[K]()
	for key, text := range staged {
		if text != nil {
			pending.Set(key, text...)
		}
	}
	committed := s.committed.SearchExcept(term, func(key K) bool {
		_, ok := staged[key]
		return ok
	})
	return search.Merge(committed, pending.Search(term)), nil
}

func (s *Search[S, K]) flush(staged texts[K]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, text := range staged {
		if text == nil {
			s.committed.Delete(key)
			continue
		}
		s.committed.Set(key, text...)
	}
}
