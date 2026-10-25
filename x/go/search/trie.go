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
	"slices"
	"unicode/utf8"
)

// node is a trie node keyed by rune. A token ends at a node when the node has postings.
type node struct {
	parent   *node
	edges    []edge
	postings []posting
}

type edge struct {
	label rune
	node  *node
}

// posting places one entry in the postings of a token. slot is the index of the
// matching term in that entry.
type posting struct {
	id   uint32
	slot uint32
}

func (n *node) child(r rune) *node {
	for _, e := range n.edges {
		if e.label == r {
			return e.node
		}
	}
	return nil
}

// find returns the node of token, or nil if no indexed token has token as a prefix.
func (n *node) find(token string) *node {
	for _, r := range token {
		if n = n.child(r); n == nil {
			return nil
		}
	}
	return n
}

func (n *node) insert(token string) *node {
	for _, r := range token {
		c := n.child(r)
		if c == nil {
			c = &node{parent: n}
			n.edges = append(n.edges, edge{label: r, node: c})
		}
		n = c
	}
	return n
}

// prune removes n and each ancestor that no longer holds a token or a child.
func (n *node) prune() {
	for n.parent != nil && len(n.postings) == 0 && len(n.edges) == 0 {
		p := n.parent
		p.edges = slices.DeleteFunc(p.edges, func(e edge) bool { return e.node == n })
		n = p
	}
}

// fuzzy appends to out every token node whose path from n is within one rune edit of
// word, skipping nodes already in out.
func (n *node) fuzzy(word string, out []*node) []*node {
	if word == "" {
		out = appendToken(out, n)
		for _, e := range n.edges {
			out = appendToken(out, e.node)
		}
		return out
	}
	r, size := utf8.DecodeRuneInString(word)
	rest := word[size:]
	if c := n.child(r); c != nil {
		out = c.fuzzy(rest, out)
	}
	out = appendToken(out, n.find(rest))
	for _, e := range n.edges {
		if e.label != r {
			out = appendToken(out, e.node.find(rest))
		}
		out = appendToken(out, e.node.find(word))
	}
	return out
}

func appendToken(out []*node, n *node) []*node {
	if n == nil || len(n.postings) == 0 || slices.Contains(out, n) {
		return out
	}
	return append(out, n)
}
