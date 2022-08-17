package gorp

// Compound wraps a slice of Retrieve queries into a set of Clauses that are meant to be
// executed together.
type Compound[K Key, E Entry[K]] struct {
	Clauses []Retrieve[K, E]
}

// Next opens a new Retrieve query, adds it to Compound.Clauses and returns it.
func (s *Compound[K, E]) Next() Retrieve[K, E] {
	n := NewRetrieve[K, E]()
	s.Clauses = append(s.Clauses, NewRetrieve[K, E]())
	return n
}

// Current returns the most recently added Retrieve query.
func (s *Compound[K, E]) Current() Retrieve[K, E] { return s.Clauses[len(s.Clauses)-1] }
