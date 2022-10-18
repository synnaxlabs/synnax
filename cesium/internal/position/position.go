package position

type Position int64

func (p Position) IsZero() bool { return p == 0 }

func (p Position) Before(other Position) bool { return p < other }

func (p Position) After(other Position) bool { return p > other }

func (p Position) BeforeEq(other Position) bool { return p <= other }

func (p Position) AfterEq(other Position) bool { return p >= other }

func (p Position) Add(span Span) Position {
	if span > 0 && p > Max-Position(span) {
		return Max
	}
	if span < 0 && p < Min-Position(span) {
		return Min
	}
	return Position(span) + p
}

func (p Position) Sub(span Span) Position { return p.Add(-span) }

func (p Position) Range(other Position) Range { return Range{p, other} }

func (p Position) SpanRange(span Span) Range {
	r := p.Range(p.Add(span))
	if !r.Valid() {
		r = r.Swap()
	}
	return r
}

func (p Position) Span(other Position) Span { return Span(uint64(other) - uint64(p)) }

const (
	Min = Position(0)
	Max = Position(int64(^uint64(0) >> 1))
)

type Range struct {
	Start Position
	End   Position
}

func (r Range) Span() Span { return r.Start.Span(r.End) }

func (r Range) BoundBy(other Range) Range {
	if other.Start.After(r.Start) {
		r.Start = other.Start
	}
	if other.Start.After(r.End) {
		r.End = other.Start
	}
	if other.End.Before(r.End) {
		r.End = other.End
	}
	if other.End.Before(r.Start) {
		r.Start = other.End
	}
	return r
}

func (r Range) ContainsPos(p Position) bool {
	return p.AfterEq(r.Start) && p.Before(r.End)
}

func (r Range) ContainsRange(other Range) bool {
	return other.Start.BeforeEq(r.Start) && other.End.AfterEq(r.End)
}

func (r Range) OverlapsWith(other Range) bool {
	if r.End == other.Start || r.Start == other.End {
		return false
	}
	return r.ContainsPos(other.End) ||
		r.ContainsPos(other.Start) ||
		other.ContainsPos(r.Start) ||
		other.ContainsPos(r.End)
}

func (r Range) Equals(other Range) bool {
	return r.Start == other.Start && r.End == other.End
}

func (r Range) IsZero() bool { return r.Span().IsZero() }

func (r Range) Swap() Range { return Range{Start: r.End, End: r.Start} }

func (r Range) Valid() bool { return r.Span() >= 0 }

func (r Range) Midpoint() Position {
	return r.Start.Add(r.Span() / 2)
}

var (
	RangeMax  = Range{Start: Min, End: Max}
	RangeZero = Range{}
)

type Span int64

func (s Span) IsZero() bool { return s == 0 }

type Bytes uint64

// Approximation is an approximate position. position. A Approximation with zero span
// indicates that the position has been resolved with certainty. A Approximation with a
// non-zero span indicates that the exact position is unknown, but that the position is
// within the range.
type Approximation struct{ Range }

func ExactlyAt(r Position) Approximation {
	return Approximation{Range: r.SpanRange(0)}
}

func Between(start Position, end Position) Approximation {
	return Approximation{Range: Range{Start: start, End: end}}
}

func Before(end Position) Approximation {
	return Between(Min, end)
}

func After(start Position) Approximation {
	return Between(start, Max)
}

var Uncertain = Approximation{Range: Range{Start: Min, End: Max}}

// Uncertainty returns a scalar value representing the confidence of the index in resolving
// the position. A value of 0 indicates that the position has been resolved with certainty.
// A value greater than 0 indicates that the exact position is unknown.
func (i Approximation) Uncertainty() Span { return i.Range.Span() }

func (i Approximation) Exact() bool { return i.Uncertainty().IsZero() }

// Value returns a best guess of the position.
func (i Approximation) Value() Position { return i.Range.Midpoint() }

func (i Approximation) Uncertain() bool {
	return i.Range.Start == Min && i.Range.End == Max
}

func (i Approximation) Contains(pos Position) bool {
	return i.Range.ContainsPos(pos)
}

func (i Approximation) MustContain(pos Position) {
	if !i.Contains(pos) {
		panic("position out of approximation")
	}
}
