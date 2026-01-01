// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// CompoundRetrieve wraps a slice of Retrieve queries into a set of Clauses that are meant to be
// executed together.
type CompoundRetrieve[K Key, E Entry[K]] struct {
	Clauses []Retrieve[K, E]
}

// Next opens a new Retrieve query, adds it to CompoundRetrieve.Clauses and returns it.
func (s *CompoundRetrieve[K, E]) Next() Retrieve[K, E] {
	n := NewRetrieve[K, E]()
	s.Clauses = append(s.Clauses, NewRetrieve[K, E]())
	return n
}

// Current returns the most recently changes Retrieve query.
func (s *CompoundRetrieve[K, E]) Current() Retrieve[K, E] { return s.Clauses[len(s.Clauses)-1] }
