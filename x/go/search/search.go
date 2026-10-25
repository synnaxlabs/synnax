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

const (
	// retainedScratches is how many idle search scratches an Index keeps.
	retainedScratches = 4
	// maxQueryWords is how many words of a term Search reads.
	maxQueryWords = 64
)

// Index is a full-text index of entries keyed by K. It is safe for concurrent use.
type Index[K comparable] struct {
	// scratches holds idle search scratches. Unlike a sync.Pool, it keeps them across
	// garbage collections, since each is sized to every entry.
	scratches chan *scratch
	// mu guards every field below.
	mu      sync.RWMutex
	trie    trie
	ids     map[K]uint32
	entries []entry[K]
	free    []uint32
}

// entry holds one term for each unique token of its key, in the sorted order of the
// tokens.
type entry[K comparable] struct {
	key   K
	terms []term
}

// term places an entry in a token node. pos is the index of the entry's posting in
// that node.
type term struct {
	node uint32
	pos  uint32
}

// scratch is the state of one search. slots maps an entry id to one plus the index of
// its match, or to 0 when the entry has not matched.
type scratch struct {
	slots   []uint32
	matches []match
	fuzzy   []uint32
	ranked  []uint64
}

// match is how one entry matches a search. word is one plus the index of the last
// query word the entry matched.
type match struct {
	id      uint32
	score   int32
	matched uint32
	word    uint32
}

// New returns an empty Index.
func New[K comparable]() *Index[K] {
	return &Index[K]{
		scratches: make(chan *scratch, retainedScratches),
		trie:      newTrie(),
		ids:       make(map[K]uint32),
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
	slices.Sort(tokens)
	tokens = slices.Compact(tokens)
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
		n := idx.trie.insert(tok)
		postings := &idx.trie.node(n).postings
		e.terms = append(e.terms, term{node: n, pos: uint32(len(*postings))})
		*postings = append(*postings, posting{id: id, slot: uint32(len(e.terms) - 1)})
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
// no entry matches every word, it returns the entries that match any word. It reads
// only the first 64 words of term, and returns at most limit entries, or every entry
// when limit is 0.
func (idx *Index[K]) Search(term string, limit int) Result[K] {
	return idx.search(term, nil, limit)
}

// SearchExcept is Search with the keys for which except returns true left out. It
// leaves them out before it checks whether any entry matches every word.
func (idx *Index[K]) SearchExcept(
	term string,
	except func(K) bool,
	limit int,
) Result[K] {
	return idx.search(term, except, limit)
}

func (idx *Index[K]) search(term string, except func(K) bool, limit int) Result[K] {
	words := tokenize(firstWords(term, maxQueryWords))
	if len(words) == 0 {
		return Result[K]{}
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	s := idx.scratch()
	if n := len(idx.entries) - len(s.slots); n > 0 {
		s.slots = append(s.slots, make([]uint32, n)...)
	}
	for i, word := range words {
		s.match(&idx.trie, word, uint32(i+1))
	}
	complete := false
	for _, m := range s.matches {
		if m.matched == uint32(len(words)) && idx.kept(m.id, except) {
			s.ranked = append(s.ranked, idx.rank(m.id, m.score))
			complete = true
		}
	}
	if !complete {
		for _, m := range s.matches {
			if idx.kept(m.id, except) {
				s.ranked = append(s.ranked, idx.rank(m.id, m.score))
			}
		}
	}
	slices.Sort(s.ranked)
	kept := s.ranked[:limited(len(s.ranked), limit)]
	res := Result[K]{
		Keys:     make([]K, len(kept)),
		ranks:    make([]uint32, len(kept)),
		complete: complete,
	}
	for i, r := range kept {
		res.Keys[i] = idx.entries[uint32(r)].key
		res.ranks[i] = uint32(r >> 32)
	}
	for _, m := range s.matches {
		s.slots[m.id] = 0
	}
	s.matches, s.ranked = s.matches[:0], s.ranked[:0]
	select {
	case idx.scratches <- s:
	default:
	}
	return res
}

// scratch returns an idle scratch, or a new one when none is idle.
func (idx *Index[K]) scratch() *scratch {
	select {
	case s := <-idx.scratches:
		return s
	default:
		return new(scratch)
	}
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
func (s *scratch) match(t *trie, word string, w uint32) {
	exact := t.find(root, word)
	if exact != none {
		s.add(t, exact, exactWeight, w)
		for _, e := range t.node(exact).edges {
			s.addTree(t, e.node, w)
		}
	}
	s.fuzzy = t.fuzzy(root, word, s.fuzzy[:0])
	for _, n := range s.fuzzy {
		// The exact node and its children already count as exact and prefix matches,
		// and the exact node can hold many postings.
		if exact == none || (n != exact && t.node(n).parent != exact) {
			s.add(t, n, fuzzyWeight, w)
		}
	}
}

func (s *scratch) addTree(t *trie, n, w uint32) {
	s.add(t, n, prefixWeight, w)
	for _, e := range t.node(n).edges {
		s.addTree(t, e.node, w)
	}
}

func (s *scratch) add(t *trie, n uint32, weight int32, w uint32) {
	for _, p := range t.node(n).postings {
		slot := s.slots[p.id]
		if slot == 0 {
			s.matches = append(s.matches, match{id: p.id})
			slot = uint32(len(s.matches))
			s.slots[p.id] = slot
		}
		if m := &s.matches[slot-1]; m.word != w {
			m.score, m.matched, m.word = m.score+weight, m.matched+1, w
		}
	}
}

// holds reports whether the entry at id holds exactly tokens, which are sorted and
// unique.
func (idx *Index[K]) holds(id uint32, tokens []string) bool {
	terms := idx.entries[id].terms
	if len(terms) != len(tokens) {
		return false
	}
	for i, tok := range tokens {
		if idx.trie.find(root, tok) != terms[i].node {
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
	for _, tm := range e.terms {
		n := idx.trie.node(tm.node)
		last := len(n.postings) - 1
		moved := n.postings[last]
		n.postings[tm.pos] = moved
		idx.entries[moved.id].terms[moved.slot].pos = tm.pos
		n.postings = n.postings[:last]
		if last == 0 {
			n.postings = nil
			idx.trie.prune(tm.node)
		}
	}
	e.terms = e.terms[:0]
}
