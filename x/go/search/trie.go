// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search

import (
	"math"
	"slices"
	"unicode/utf8"
)

// trie is a rune trie whose nodes live in pages and refer to each other by index, so
// the garbage collector does not trace them. A token ends at a node when the node has
// postings.
type trie struct {
	// pages holds the nodes. The first page grows with the trie, so a small trie stays
	// small. Every later page is allocated full and never moves.
	pages [][]node
	len   uint32
	// free holds the indexes of pruned nodes, for reuse.
	free []uint32
}

const (
	// root is the index of the root node.
	root uint32 = 0
	// none is the index that child and find return when no node exists.
	none uint32 = math.MaxUint32
	// pageBits sets the maximum number of nodes in a page to 1<<pageBits.
	pageBits = 8
	pageMask = 1<<pageBits - 1
)

type node struct {
	parent   uint32
	edges    []edge
	postings []posting
}

type edge struct {
	label rune
	node  uint32
}

// posting places one entry in the postings of a token. slot is the index of the
// matching term in that entry.
type posting struct {
	id   uint32
	slot uint32
}

func newTrie() trie {
	t := trie{}
	t.alloc(root)
	return t
}

// node returns the node at index n. The pointer is valid until the next alloc.
func (t *trie) node(n uint32) *node { return &t.pages[n>>pageBits][n&pageMask] }

func (t *trie) child(n uint32, r rune) uint32 {
	for _, e := range t.node(n).edges {
		if e.label == r {
			return e.node
		}
	}
	return none
}

// find returns the node of token below n, or none if no indexed token below n has token
// as a prefix.
func (t *trie) find(n uint32, token string) uint32 {
	for _, r := range token {
		if n = t.child(n, r); n == none {
			return none
		}
	}
	return n
}

func (t *trie) insert(token string) uint32 {
	n := root
	for _, r := range token {
		c := t.child(n, r)
		if c == none {
			c = t.alloc(n)
			parent := t.node(n)
			parent.edges = append(parent.edges, edge{label: r, node: c})
		}
		n = c
	}
	return n
}

func (t *trie) alloc(parent uint32) uint32 {
	if k := len(t.free); k > 0 {
		n := t.free[k-1]
		t.free = t.free[:k-1]
		t.node(n).parent = parent
		return n
	}
	n := t.len
	page := n >> pageBits
	if page == uint32(len(t.pages)) {
		capacity := 1 << pageBits
		if page == 0 {
			capacity = 0
		}
		t.pages = append(t.pages, make([]node, 0, capacity))
	}
	t.pages[page] = append(t.pages[page], node{parent: parent})
	t.len++
	return n
}

// prune frees n and each ancestor that no longer holds a token or a child.
func (t *trie) prune(n uint32) {
	for n != root && len(t.node(n).postings) == 0 && len(t.node(n).edges) == 0 {
		p := t.node(n).parent
		parent := t.node(p)
		parent.edges = slices.DeleteFunc(
			parent.edges,
			func(e edge) bool { return e.node == n },
		)
		t.free = append(t.free, n)
		n = p
	}
}

// fuzzy appends to out every token node whose path from n is within one rune edit of
// word, skipping nodes already in out.
func (t *trie) fuzzy(n uint32, word string, out []uint32) []uint32 {
	if word == "" {
		out = t.appendToken(out, n)
		for _, e := range t.node(n).edges {
			out = t.appendToken(out, e.node)
		}
		return out
	}
	r, size := utf8.DecodeRuneInString(word)
	rest := word[size:]
	if c := t.child(n, r); c != none {
		out = t.fuzzy(c, rest, out)
	}
	out = t.appendToken(out, t.find(n, rest))
	for _, e := range t.node(n).edges {
		if e.label != r {
			out = t.appendToken(out, t.find(e.node, rest))
		}
		out = t.appendToken(out, t.find(e.node, word))
	}
	return out
}

func (t *trie) appendToken(out []uint32, n uint32) []uint32 {
	if n == none || len(t.node(n).postings) == 0 || slices.Contains(out, n) {
		return out
	}
	return append(out, n)
}
