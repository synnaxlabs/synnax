// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package search is an in-memory full-text index. It matches each word of a query
// against the tokens of an entry exactly, by prefix, or within one rune edit.
package search

import (
	"slices"
	"sync"
)

// The match weights rank an exact word far above a prefix, and a prefix above a word
// within one edit.
const (
	exactWeight  = 100
	prefixWeight = 10
	fuzzyWeight  = 1
)

// Index is a full-text index of entries keyed by K. It is safe for concurrent use.
type Index[K comparable] struct {
	// pool holds scratch values for Search, each sized to entries.
	pool sync.Pool
	// mu guards every field below.
	mu      sync.RWMutex
	root    node
	ids     map[K]uint32
	entries []entry[K]
	free    []uint32
}

// entry holds one term for each unique token of its key.
type entry[K comparable] struct {
	key   K
	terms []term
}

// term places an entry in a token node. pos is the index of the entry's posting in
// that node.
type term struct {
	node *node
	pos  uint32
}

// scratch is the state of one search. state is indexed by entry id, and touched lists
// the ids whose state is not zero.
type scratch struct {
	state   []match
	touched []uint32
	fuzzy   []*node
	ranked  []uint64
}

// match is how one entry matches a search. word is one plus the index of the last
// query word the entry matched.
type match struct {
	score   int32
	matched uint32
	word    uint32
}

// New returns an empty Index.
func New[K comparable]() *Index[K] {
	return &Index[K]{
		pool: sync.Pool{New: func() any { return new(scratch) }},
		ids:  make(map[K]uint32),
	}
}

// Set indexes key under the tokens of text, replacing the tokens it held before.
func (idx *Index[K]) Set(key K, text ...string) {
	var tokens []string
	if len(text) == 1 {
		tokens = tokenize(text[0])
	} else {
		for _, t := range text {
			tokens = append(tokens, tokenize(t)...)
		}
	}
	unique := tokens[:0]
	for _, tok := range tokens {
		if !slices.Contains(unique, tok) {
			unique = append(unique, tok)
		}
	}
	tokens = unique
	idx.mu.Lock()
	defer idx.mu.Unlock()
	id, ok := idx.ids[key]
	if ok {
		if idx.holds(id, tokens) {
			return
		}
		idx.unlink(id)
	} else {
		id = idx.alloc(key)
	}
	e := &idx.entries[id]
	e.terms = slices.Grow(e.terms, len(tokens))
	for _, tok := range tokens {
		n := idx.root.insert(tok)
		e.terms = append(e.terms, term{node: n, pos: uint32(len(n.postings))})
		n.postings = append(n.postings, posting{id: id, slot: uint32(len(e.terms) - 1)})
	}
}

// Delete removes key from the index. It does nothing if key is not indexed.
func (idx *Index[K]) Delete(key K) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	id, ok := idx.ids[key]
	if !ok {
		return
	}
	idx.unlink(id)
	delete(idx.ids, key)
	var zero K
	idx.entries[id].key = zero
	idx.free = append(idx.free, id)
}

// Search returns the entries that match every word of term, closest match first. When
// no entry matches every word, it returns the entries that match any word.
func (idx *Index[K]) Search(term string) Result[K] { return idx.search(term, nil) }

// SearchExcept is Search with the keys for which except returns true left out. It
// leaves them out before it checks whether any entry matches every word.
func (idx *Index[K]) SearchExcept(term string, except func(K) bool) Result[K] {
	return idx.search(term, except)
}

func (idx *Index[K]) search(term string, except func(K) bool) Result[K] {
	words := tokenize(term)
	if len(words) == 0 {
		return Result[K]{}
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	s := idx.pool.Get().(*scratch)
	if n := len(idx.entries) - len(s.state); n > 0 {
		s.state = append(s.state, make([]match, n)...)
	}
	for i, word := range words {
		s.match(&idx.root, word, uint32(i+1))
	}
	complete := false
	for _, id := range s.touched {
		if s.state[id].matched == uint32(len(words)) && idx.kept(id, except) {
			s.ranked = append(s.ranked, idx.rank(id, s.state[id].score))
			complete = true
		}
	}
	if !complete {
		for _, id := range s.touched {
			if idx.kept(id, except) {
				s.ranked = append(s.ranked, idx.rank(id, s.state[id].score))
			}
		}
	}
	slices.Sort(s.ranked)
	res := Result[K]{
		Keys:     make([]K, len(s.ranked)),
		ranks:    make([]uint32, len(s.ranked)),
		complete: complete,
	}
	for i, r := range s.ranked {
		res.Keys[i] = idx.entries[uint32(r)].key
		res.ranks[i] = uint32(r >> 32)
	}
	for _, id := range s.touched {
		s.state[id] = match{}
	}
	s.touched, s.ranked = s.touched[:0], s.ranked[:0]
	idx.pool.Put(s)
	return res
}

func (idx *Index[K]) kept(id uint32, except func(K) bool) bool {
	return except == nil || !except(idx.entries[id].key)
}

// rank packs the rank of an entry in its high 32 bits and its id in its low 32 bits,
// so the packed values sort by rank, then by id.
func (idx *Index[K]) rank(id uint32, score int32) uint64 {
	return uint64(newRank(score, len(idx.entries[id].terms)))<<32 | uint64(id)
}

// match adds the entries that match word, the word at index w-1 of the query. It adds
// exact matches before prefixes, and prefixes before fuzzy matches, so the first match
// of an entry for a word is its best.
func (s *scratch) match(root *node, word string, w uint32) {
	exact := root.find(word)
	if exact != nil {
		s.add(exact, exactWeight, w)
		for _, e := range exact.edges {
			s.addTree(e.node, w)
		}
	}
	s.fuzzy = root.fuzzy(word, s.fuzzy[:0])
	for _, n := range s.fuzzy {
		// The exact node and its children already count as exact and prefix matches,
		// and the exact node can hold many postings.
		if exact == nil || (n != exact && n.parent != exact) {
			s.add(n, fuzzyWeight, w)
		}
	}
	clear(s.fuzzy)
}

func (s *scratch) addTree(n *node, w uint32) {
	s.add(n, prefixWeight, w)
	for _, e := range n.edges {
		s.addTree(e.node, w)
	}
}

func (s *scratch) add(n *node, weight int32, w uint32) {
	for _, p := range n.postings {
		m := &s.state[p.id]
		if m.word == w {
			continue
		}
		if m.matched == 0 {
			s.touched = append(s.touched, p.id)
		}
		m.score, m.matched, m.word = m.score+weight, m.matched+1, w
	}
}

// holds reports whether the entry at id holds exactly tokens.
func (idx *Index[K]) holds(id uint32, tokens []string) bool {
	terms := idx.entries[id].terms
	if len(terms) != len(tokens) {
		return false
	}
	for _, tok := range tokens {
		n := idx.root.find(tok)
		if n == nil ||
			!slices.ContainsFunc(terms, func(t term) bool { return t.node == n }) {
			return false
		}
	}
	return true
}

func (idx *Index[K]) alloc(key K) uint32 {
	var id uint32
	if n := len(idx.free); n > 0 {
		id, idx.free = idx.free[n-1], idx.free[:n-1]
	} else {
		id = uint32(len(idx.entries))
		idx.entries = append(idx.entries, entry[K]{})
	}
	idx.entries[id].key = key
	idx.ids[key] = id
	return id
}

// unlink removes every posting of the entry at id. Each removal moves the last posting
// of the node into the freed position, so it runs in constant time.
func (idx *Index[K]) unlink(id uint32) {
	e := &idx.entries[id]
	for _, t := range e.terms {
		n := t.node
		last := len(n.postings) - 1
		moved := n.postings[last]
		n.postings[t.pos] = moved
		idx.entries[moved.id].terms[moved.slot].pos = t.pos
		n.postings = n.postings[:last]
		if last == 0 {
			n.postings = nil
			n.prune()
		}
	}
	clear(e.terms)
	e.terms = e.terms[:0]
}
